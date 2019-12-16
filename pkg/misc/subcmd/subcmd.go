package subcmd

import (
	"errors"
	"os"
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
	c.statMu.Lock()
	defer c.statMu.Unlock()
	return c.status == StatusStarting || c.status == StatusStarted
}

type Subcmd struct {
	*exec.Cmd

	Setup       func(*exec.Cmd) error
	MaxRestarts int

	Started chan *exec.Cmd

	callbacks []func(*Subcmd)

	current *exec.Cmd
	status  Status

	lastErr    error
	lastStatus int
	restarts   int

	waitCh chan error

	cbMu   sync.Mutex
	currMu sync.Mutex
	statMu sync.Mutex
	waitMu sync.Mutex
}

func New(name string, arg ...string) *Subcmd {
	s := &Subcmd{
		Cmd:         exec.Command(name, arg...),
		MaxRestarts: -1,
		status:      StatusStopped,
	}
	return s
}

func (sc *Subcmd) OnStatusChange(cb func(*Subcmd)) {
	cb(sc)
	sc.cbMu.Lock()
	sc.callbacks = append(sc.callbacks, cb)
	sc.cbMu.Unlock()
}

func (sc *Subcmd) Status() Status {
	sc.statMu.Lock()
	defer sc.statMu.Unlock()
	return sc.status
}

func (sc *Subcmd) Start() error {
	if sc.Status() == StatusStarting || sc.Status() == StatusStarted {
		return errors.New("already started")
	}
	return sc.start()
}

func (sc *Subcmd) Restart() error {
	if sc.Status() == StatusStarting {
		return errors.New("already starting")
	}
	if !Running(sc) {
		return sc.start()
	}
	return sc.terminate(false)
}

func (sc *Subcmd) Stop() error {
	if !Running(sc) {
		return errors.New("not running")
	}
	return sc.terminate(true)
}

func (sc *Subcmd) terminate(stop bool) error {
	if stop {
		defer sc.setStatus(StatusStopped)
	}
	syscall.Kill(-sc.current.Process.Pid, syscall.SIGTERM)
	process, err := os.FindProcess(sc.current.Process.Pid)
	if err != nil {
		return nil
	}
	syscall.Kill(-process.Pid, syscall.SIGKILL)
	return nil
}

func (sc *Subcmd) Wait() error {
	sc.waitMu.Lock()
	if sc.waitCh != nil {
		sc.waitMu.Unlock()
		return errors.New("wait already called")
	}
	sc.waitCh = make(chan error)
	sc.waitMu.Unlock()
	return <-sc.waitCh
}

func (sc *Subcmd) setStatus(s Status) {
	sc.statMu.Lock()
	if sc.status == s {
		sc.statMu.Unlock()
		return
	}
	sc.status = s
	sc.statMu.Unlock()
	sc.cbMu.Lock()
	for _, cb := range sc.callbacks {
		cb(sc)
	}
	sc.cbMu.Unlock()
}

func (sc *Subcmd) Error() error {
	return sc.lastErr
}

func (sc *Subcmd) start() (err error) {
	sc.currMu.Lock()
	sc.setStatus(StatusStarting)

	sc.current = &exec.Cmd{
		Path:        sc.Cmd.Path,
		Args:        sc.Cmd.Args,
		Env:         sc.Cmd.Env,
		Dir:         sc.Cmd.Dir,
		ExtraFiles:  sc.Cmd.ExtraFiles,
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
	}

	if sc.Setup != nil {
		if err := sc.Setup(sc.current); err != nil {
			sc.currMu.Unlock()
			return err
		}
	}

	err = sc.current.Start()
	if err != nil {
		sc.setStatus(StatusStopped)
		sc.currMu.Unlock()
		return err
	}

	go func() {
		// process died too quickly?
		if sc.current.Process == nil {
			sc.setStatus(StatusStopped)
			sc.currMu.Unlock()
			return
		}

		sc.setStatus(StatusStarted)
		if sc.Started != nil {
			sc.Started <- sc.current
		}

		sc.lastErr = sc.current.Wait()
		sc.lastStatus = exitStatus(sc.lastErr)
		if sc.Status() != StatusStopped {
			sc.setStatus(StatusExited)
		}

		sc.waitMu.Lock()
		if sc.waitCh != nil {
			sc.waitCh <- sc.lastErr
			sc.waitCh = nil
		}
		sc.waitMu.Unlock()

		if sc.lastErr != nil && sc.lastStatus != -1 {
			sc.currMu.Unlock()
			return
		}
		sc.currMu.Unlock()

		if sc.MaxRestarts >= 0 && sc.restarts >= sc.MaxRestarts {
			return
		}

		if sc.Status() != StatusStopped {
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
