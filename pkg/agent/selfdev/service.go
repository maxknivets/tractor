package selfdev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/agent/logger"
	"github.com/manifold/tractor/pkg/daemon"
	"github.com/manifold/tractor/pkg/log"
)

type Service struct {
	Agent   *agent.Agent
	Daemon  *daemon.Daemon
	Logger  log.DebugLogger
	Console *logger.Service

	watcher *fsnotify.Watcher
}

func (s *Service) InitializeDaemon() (err error) {
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	// for typescript files
	for _, path := range collectDirs("./extension", []string{"node_modules", "out"}) {
		err = s.watcher.Add(path)
		if err != nil {
			return err
		}
	}
	// for go files
	for _, path := range collectDirs("./pkg", nil) {
		err = s.watcher.Add(path)
		if err != nil {
			return err
		}
	}
	return err
}

func (s *Service) TerminateDaemon() error {
	return s.watcher.Close()
}

func (s *Service) Serve(ctx context.Context) {
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

			if filepath.Ext(event.Name) == ".ts" {
				s.Logger.Debug("ts file changed, compiling...")
				go func() {
					cmd := exec.Command("tsc", "-p", "./extension")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Run()
					s.Logger.Debug("finished")
				}()
			}

			if filepath.Ext(event.Name) == ".go" {
				s.Logger.Debug("go file changed, testing/compiling...")
				errs := make(chan error)
				go func() {
					cmd := exec.Command("go", "build", "-o", "./dev/bin/tractor.tmp", "./cmd/tractor")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					errs <- err
					if exitStatus(err) > 0 {
						s.Logger.Debug("ERROR")
					} else {
						s.Logger.Debug("finished")
					}
				}()
				go func() {
					cmd := exec.Command("go", "build", "-o", "./dev/bin/tractor-agent.tmp", "./cmd/tractor-agent")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					errs <- err
					if exitStatus(err) > 0 {
						s.Logger.Debug("ERROR")
					} else {
						s.Logger.Debug("finished")
					}
				}()
				go func() {
					cmd := exec.Command("go", "test", "./pkg/...")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					errs <- err
					if exitStatus(err) > 0 {
						s.Logger.Debug("ERROR")
					} else {
						s.Logger.Debug("finished")
					}
				}()
				go func() {
					for i := 0; i < 3; i++ {
						if err := <-errs; err != nil {
							os.Remove("./dev/bin/tractor-agent.tmp")
							os.Remove("./dev/bin/tractor.tmp")
							return
						}
					}
					os.Rename("./dev/bin/tractor-agent.tmp", "./dev/bin/tractor-agent")
					os.Rename("./dev/bin/tractor.tmp", "./dev/bin/tractor")
					s.Daemon.OnFinished = func() {
						fmt.Println("OK EXEC ME")
					}
					s.Daemon.Terminate()
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
