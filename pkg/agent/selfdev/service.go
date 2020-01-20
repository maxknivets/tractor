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

	"github.com/fsnotify/fsnotify"
	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/agent/console"
	"github.com/manifold/tractor/pkg/misc/daemon"
	"github.com/manifold/tractor/pkg/misc/logging"
	"github.com/manifold/tractor/pkg/misc/subcmd"
)

type Service struct {
	Agent   *agent.Agent
	Daemon  *daemon.Daemon
	Logger  logging.DebugLogger
	Console *console.Service

	watcher *fsnotify.Watcher
	output  io.WriteCloser
}

func (s *Service) InitializeDaemon() (err error) {
	s.output = s.Console.NewPipe("dev")
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, path := range collectDirs("./pkg", nil) {
		err = s.watcher.Add(path)
		if err != nil {
			return err
		}
	}
	for _, path := range collectDirs("./lib", nil) {
		err = s.watcher.Add(path)
		if err != nil {
			return err
		}
	}

	for _, path := range collectDirs("./studio/extension/src", []string{"node_modules", "out"}) {
		err = s.watcher.Add(path)
		if err != nil {
			return err
		}
	}

	for _, path := range collectDirs("./studio/plugins", []string{"node_modules", "out"}) {
		err = s.watcher.Add(path)
		if err != nil {
			return err
		}
	}

	shellBuild := &cmdService{
		subcmd.New("yarn", "run", "theia", "build", "--watch", "--mode", "development"),
	}
	shellBuild.Setup = func(cmd *exec.Cmd) error {
		cmd.Dir = "./studio/shell"
		cmd.Stdout = s.output
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
	return s.watcher.Close()
}

func (s *Service) Serve(ctx context.Context) {
	debounce := Debounce(20 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			if filepath.Ext(event.Name) == ".ts" || filepath.Ext(event.Name) == ".tsx" {
				debounce(func() {
					s.Logger.Debug("ts file changed, compiling...")
					for _, plugin := range []string{"inspector", "moduleview", "tableview"} {
						if strings.Contains(event.Name, "/studio/plugins/"+plugin) {
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
					// if strings.Contains(event.Name, "/studio/extension/") {
					go func() {
						// theia extension
						cmd := exec.Command("tsc", "-p", "./studio/extension")
						cmd.Stdout = s.output
						cmd.Stderr = s.output
						cmd.Run()
						s.Logger.Debug("finished")
					}()
					// }
				})

			}

			if filepath.Ext(event.Name) == ".go" {
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

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.Logger.Debug("error:", err)
		}
	}
}

func collectDirs(path string, ignoreNames []string) []string {
	var dirs []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			for _, name := range ignoreNames {
				if info.Name() == name {
					return filepath.SkipDir
				}
			}
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return dirs
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
