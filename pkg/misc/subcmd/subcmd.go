package subcmd

import (
	"errors"
	"os/exec"
	"sync"
	"syscall"

	"github.com/manifold/tractor/pkg/data/icons"
)

type Status string

const (
	StatusStarting Status = "Starting"
	StatusStarted  Status = "Started"
	StatusExited   Status = "Exited"
	StatusStopped  Status = "Stopped"
)

func (s Status) Icon() []byte {
	switch s {
	case StatusStarted:
		return icons.Available
	case StatusStopped:
		return icons.Unavailable
	default:
		return icons.Partially
	}
}

func (s Status) String() string {
	return string(s)
}

func Running(c *Subcmd) bool {
	return c.Status == StatusStarting || c.Status == StatusStarted
}

type Subcmd struct {
	*exec.Cmd

	Setup       func(*exec.Cmd) error
	MaxRestarts int
	Status      Status
	Started     chan *exec.Cmd

	callbacks []func(*Subcmd)
	current   *exec.Cmd

	lastErr    error
	lastStatus int
	restarts   int

	waitCh chan error

	mu sync.Mutex
}

func New(name string, arg ...string) *Subcmd {
	s := &Subcmd{
		Cmd:         exec.Command(name, arg...),
		MaxRestarts: -1,
		Status:      StatusStopped,
	}
	return s
}

func (sc *Subcmd) OnStatusChange(cb func(*Subcmd)) {
	cb(sc)
	sc.mu.Lock()
	sc.callbacks = append(sc.callbacks, cb)
	sc.mu.Unlock()
}

func (sc *Subcmd) Start() error {
	if sc.Status == StatusStarting || sc.Status == StatusStarted {
		return errors.New("already started")
	}
	return sc.start()
}

func (sc *Subcmd) Restart() error {
	if sc.Status == StatusStarting {
		return errors.New("already starting")
	}
	if sc.current != nil {
		syscall.Kill(-sc.current.Process.Pid, syscall.SIGTERM)
	}
	return sc.start()
}

func (sc *Subcmd) Stop() error {
	if sc.current == nil {
		return errors.New("not running")
	}
	sc.setStatus(StatusStopped)
	return syscall.Kill(-sc.current.Process.Pid, syscall.SIGTERM)
}

func (sc *Subcmd) Wait() error {
	if sc.waitCh != nil {
		return errors.New("wait already called")
	}
	sc.waitCh = make(chan error)
	return <-sc.waitCh
}

func (sc *Subcmd) setStatus(s Status) {
	if sc.Status == s {
		return
	}
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.Status = s
	for _, cb := range sc.callbacks {
		cb(sc)
	}
}

func (sc *Subcmd) Error() error {
	return sc.lastErr
}

func (sc *Subcmd) start() (err error) {
	sc.setStatus(StatusStarting)

	sc.current = &exec.Cmd{
		Path:        sc.Cmd.Path,
		Args:        sc.Cmd.Args,
		Env:         sc.Cmd.Env,
		Dir:         sc.Cmd.Dir,
		ExtraFiles:  sc.Cmd.ExtraFiles,
		SysProcAttr: sc.Cmd.SysProcAttr,
	}

	if sc.Setup != nil {
		if err := sc.Setup(sc.current); err != nil {
			return err
		}
	}

	err = sc.current.Start()
	if err != nil {
		sc.setStatus(StatusStopped)
		return err
	}

	go func() {
		sc.setStatus(StatusStarted)
		if sc.Started != nil {
			sc.Started <- sc.current
		}

		sc.lastErr = sc.current.Wait()
		sc.lastStatus = exitStatus(sc.lastErr)
		if sc.Status != StatusStopped {
			sc.setStatus(StatusExited)
		}

		if sc.waitCh != nil {
			sc.waitCh <- sc.lastErr
			sc.waitCh = nil
		}

		if sc.lastErr != nil {
			return
		}
		sc.current = nil

		if sc.MaxRestarts >= 0 && sc.restarts >= sc.MaxRestarts {
			sc.setStatus(StatusStopped)
			return
		}

		if sc.Status != StatusStopped {
			if err := sc.start(); err != nil {
				panic(err)
			}
			sc.restarts++
		}
	}()

	return nil
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
