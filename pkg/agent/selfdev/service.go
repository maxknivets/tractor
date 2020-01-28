package selfdev

import (
	"bytes"
	"context"
	"crypto/md5"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/agent/console"
	"github.com/manifold/tractor/pkg/misc/daemon"
	"github.com/manifold/tractor/pkg/misc/logging"
	"github.com/manifold/tractor/pkg/misc/subcmd"
	"github.com/radovskyb/watcher"
)

const WatchInterval = time.Millisecond * 50

type Service struct {
	Agent   *agent.Agent
	Daemon  *daemon.Daemon
	Logger  logging.DebugLogger
	Console *console.Service

	watcher *watcher.Watcher
	output  io.WriteCloser
}

func (s *Service) InitializeDaemon() (err error) {
	s.output = s.Console.NewPipe("dev")
	s.watcher = watcher.New()
	s.watcher.SetMaxEvents(1)
	s.watcher.IgnoreHiddenFiles(true)
	s.watcher.AddFilterHook(func(info os.FileInfo, fullPath string) error {
		allowedExt := []string{".go", ".ts", ".tsx", ".js", ".jsx", ".html"}
		ignoreSubstr := []string{"node_modules"}
		for _, substr := range ignoreSubstr {
			if strings.Contains(fullPath, substr) {
				return watcher.ErrSkip
			}
		}
		for _, ext := range allowedExt {
			if filepath.Ext(info.Name()) == ext {
				return nil
			}
		}
		return watcher.ErrSkip
	})

	s.watcher.AddRecursive("./pkg")
	s.watcher.AddRecursive("./studio")

	shellBuild := &cmdService{
		subcmd.New("yarn", "run", "theia", "build", "--watch", "--mode", "development"),
	}
	shellBuild.Setup = func(cmd *exec.Cmd) error {
		cmd.Dir = "./studio/shell"
		// theia watch barfs a lot of useless warnings with every change
		//cmd.Stdout = s.output
		cmd.Stderr = s.output
		return nil
	}
	s.Daemon.AddServices(shellBuild)

	shellRun := &cmdService{
		subcmd.New("yarn", "run", "theia", "start", "--log-level", "warn", "--plugins", "local-dir:../plugins/"),
	}
	shellRun.Setup = func(cmd *exec.Cmd) error {
		cmd.Dir = "./studio/shell"
		cmd.Stdout = s.output
		cmd.Stderr = s.output
		return nil
	}
	s.Daemon.AddServices(shellRun)

	return err
}

func (s *Service) TerminateDaemon() error {
	s.watcher.Close()
	return nil
}

func (s *Service) Serve(ctx context.Context) {
	go s.handleLoop(ctx)
	if err := s.watcher.Start(WatchInterval); err != nil {
		s.Logger.Debug(err)
	}
}
func (s *Service) handleLoop(ctx context.Context) {
	// debounce := Debounce(20 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-s.watcher.Event:
			if !ok {
				return
			}
			if event.Op&watcher.Chmod == watcher.Chmod {
				continue
			}

			if filepath.Ext(event.Path) == ".ts" || filepath.Ext(event.Path) == ".tsx" {
				// debounce(func() {
				s.Logger.Debug("ts file changed, compiling...")
				for _, plugin := range []string{"inspector", "moduleview", "tableview"} {
					if strings.Contains(event.Path, "/studio/plugins/"+plugin) {
						go func() {
							// theia plugin
							cmd := exec.Command("tsc", "-p", "./studio/plugins/"+plugin)
							cmd.Stdout = s.output
							cmd.Stderr = s.output
							cmd.Run()
							s.Logger.Debug("finished")
						}()
					}
				}
				if strings.Contains(event.Path, "/studio/extension/") {
					go func() {
						// theia extension
						cmd := exec.Command("tsc", "-p", "./studio/extension")
						cmd.Stdout = s.output
						cmd.Stderr = s.output
						cmd.Run()
						s.Logger.Debug("finished")
					}()
				}
				// })

			}

			if filepath.Ext(event.Path) == ".go" {
				s.Logger.Debug("go file changed, testing/compiling...")
				errs := make(chan error)
				go func() {
					cmd := exec.Command("go", "build", "-o", "./local/bin/tractor.tmp", "./cmd/tractor")
					cmd.Stdout = s.output
					cmd.Stderr = s.output
					err := cmd.Run()
					errs <- err
					if exitStatus(err) > 0 {
						s.Logger.Debug("ERROR")
					}
				}()
				go func() {
					cmd := exec.Command("go", "build", "-o", "./local/bin/tractor-agent.tmp", "./cmd/tractor-agent")
					cmd.Stdout = s.output
					cmd.Stderr = s.output
					err := cmd.Run()
					errs <- err
					if exitStatus(err) > 0 {
						s.Logger.Debug("ERROR")
					}
				}()
				go func() {
					cmd := exec.Command("go", "test", "-race", "./pkg/...")
					cmd.Stdout = s.output
					cmd.Stderr = s.output
					err := cmd.Run()
					errs <- err
					if exitStatus(err) > 0 {
						s.Logger.Debug("ERROR")
					}
				}()
				go func() {
					for i := 0; i < 3; i++ {
						if err := <-errs; err != nil {
							os.Remove("./local/bin/tractor-agent.tmp")
							os.Remove("./local/bin/tractor.tmp")
							return
						}
					}
					os.Rename("./local/bin/tractor.tmp", "./local/bin/tractor")

					// NOTE: this is useless since go doesn't make deterministic builds.
					// 		 just a reminder maybe someday we can restart more intelligently.
					if !checksumMatch("./local/bin/tractor-agent.tmp", "./local/bin/tractor-agent") {
						os.Rename("./local/bin/tractor-agent.tmp", "./local/bin/tractor-agent")
						s.Daemon.OnFinished = func() {
							err := syscall.Exec(os.Args[0], os.Args, os.Environ())
							if err != nil {
								panic(err)
							}
						}
						s.Daemon.Terminate()
					} else {
						os.Remove("./local/bin/tractor-agent.tmp")
					}

				}()
			}

		case err, ok := <-s.watcher.Error:
			if !ok {
				return
			}
			s.Logger.Debug("error:", err)
		case <-s.watcher.Closed:
			return
		}
	}
}

func exitStatus(err error) int {
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 0
}

func checksumMatch(bin1, bin2 string) bool {
	checksum := func(path string, ch chan []byte) {
		b, err := os.Open(path)
		if err != nil {
			return
		}
		defer b.Close()
		h := md5.New()
		io.Copy(h, b)
		ch <- h.Sum(nil)
	}
	chk1 := make(chan []byte)
	chk2 := make(chan []byte)
	go checksum(bin1, chk1)
	go checksum(bin2, chk2)
	return bytes.Equal(<-chk1, <-chk2)
}

// New returns a debounced function that takes another functions as its argument.
// This function will be called when the debounced function stops being called
// for the given duration.
// The debounced function can be invoked with different functions, if needed,
// the last one will win.
func Debounce(after time.Duration) func(f func()) {
	d := &debouncer{after: after}

	return func(f func()) {
		d.add(f)
	}
}

type debouncer struct {
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
}

func (d *debouncer) add(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.after, f)
}

type cmdService struct {
	*subcmd.Subcmd
}

func (s *cmdService) Serve(ctx context.Context) {
	s.Start()
	s.Wait()
}

func (s *cmdService) TerminateDaemon() error {
	return s.Stop()
}
