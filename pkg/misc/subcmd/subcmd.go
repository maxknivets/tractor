package subcmd

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/keybase/go-ps"
	"github.com/manifold/tractor/pkg/data/icons"
	"github.com/manifold/tractor/pkg/misc/logging"
)

type Status string

const (
	StatusStarting Status = "Starting"
	StatusStarted  Status = "Started"
	StatusExited   Status = "Exited"
	StatusStopped  Status = "Stopped"
)

var (
	ErrStarted    = errors.New("already started")
	ErrStarting   = errors.New("already starting")
	ErrNotRunning = errors.New("not running")
	ErrWaiting    = errors.New("already waiting")
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

type Observer func(*Subcmd, Status)

type Subcmd struct {
	*exec.Cmd

	Setup       func(*exec.Cmd) error
	maxRestarts int

	Log logging.DebugLogger

	Started chan *exec.Cmd

	callbacks []Observer

	current *exec.Cmd
	status  Status

	lastErr    error
	lastStatus int
	restarts   int

	waitCh chan error

	cbMu      sync.Mutex
	runMu     sync.Mutex
	statMu    sync.Mutex
	waitMu    sync.Mutex
	pidMu     sync.Mutex
	lastMu    sync.Mutex
	restartMu sync.Mutex
}

func New(name string, arg ...string) *Subcmd {
	return &Subcmd{
		Cmd:         exec.Command(name, arg...),
		maxRestarts: -1,
		status:      StatusStopped,
	}
}

func (sc *Subcmd) SetMaxRestarts(n int) {
	sc.restartMu.Lock()
	defer sc.restartMu.Unlock()
	sc.maxRestarts = n
}

func (sc *Subcmd) Observe(cb Observer) {
	sc.cbMu.Lock()
	sc.callbacks = append(sc.callbacks, cb)
	sc.cbMu.Unlock()
}

// don't call in observer callback!
func (sc *Subcmd) Status() Status {
	sc.statMu.Lock()
	defer sc.statMu.Unlock()
	return sc.status
}

func (sc *Subcmd) Start() error {
	if sc.Status() == StatusStarting || sc.Status() == StatusStarted {
		return ErrStarted
	}
	return sc.start()
}

func (sc *Subcmd) Restart() error {
	if sc.Status() == StatusStarting {
		return ErrStarting
	}
	if !Running(sc) {
		return sc.start()
	}
	return sc.terminate()
}

func (sc *Subcmd) Stop() error {
	if !Running(sc) {
		return ErrNotRunning
	}
	sc.setStatus(StatusStopped)
	return sc.terminate()
}

func (sc *Subcmd) Signal(sig os.Signal) {
	sc.pidMu.Lock()
	sc.current.Process.Signal(sig)
	sc.pidMu.Unlock()
}

func (sc *Subcmd) terminate() error {
	sc.pidMu.Lock()
	if sc.current.Process == nil {
		sc.pidMu.Unlock()
		return ErrNotRunning
	}
	pid := sc.current.Process.Pid
	sc.pidMu.Unlock()
	logging.Debug(sc.Log, "sending SIGINT to PID ", pid)
	syscall.Kill(-pid, syscall.SIGINT)
	timeout := time.After(3 * time.Second) // TODO: configurable
	for {
		select {
		case <-timeout:
			logging.Debug(sc.Log, "sending SIGKILL to PID ", pid)
			syscall.Kill(-pid, syscall.SIGKILL)
			return nil
		default:
			process, _ := ps.FindProcess(pid)
			if process == nil {
				return nil
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func (sc *Subcmd) Wait() error {
	sc.waitMu.Lock()
	if sc.waitCh != nil {
		sc.waitMu.Unlock()
		return ErrWaiting
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
	logging.Debug(sc.Log, sc.status, "=>", s)
	sc.status = s
	sc.cbMu.Lock()
	for _, cb := range sc.callbacks {
		cb(sc, s)
	}
	sc.cbMu.Unlock()
	sc.statMu.Unlock()
}

func (sc *Subcmd) Error() error {
	sc.lastMu.Lock()
	defer sc.lastMu.Unlock()
	return sc.lastErr
}

func (sc *Subcmd) ExitStatus() int {
	sc.lastMu.Lock()
	defer sc.lastMu.Unlock()
	return sc.lastStatus
}

func (sc *Subcmd) start() (err error) {
	startErr := make(chan error)
	go func() {
		sc.runMu.Lock()
		sc.setStatus(StatusStarting)

		sc.pidMu.Lock()
		sc.current = &exec.Cmd{
			Path:        sc.Cmd.Path,
			Args:        sc.Cmd.Args,
			Env:         sc.Cmd.Env,
			Dir:         sc.Cmd.Dir,
			ExtraFiles:  sc.Cmd.ExtraFiles,
			SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
		}
		sc.pidMu.Unlock()

		if sc.Setup != nil {
			if err := sc.Setup(sc.current); err != nil {
				sc.runMu.Unlock()
				startErr <- err
				return
			}
		}

		err = sc.current.Start()
		if err != nil {
			sc.setStatus(StatusStopped)
			sc.runMu.Unlock()
			startErr <- err
			return
		}

		startErr <- nil

		// process died too quickly?
		if sc.current.Process == nil {
			sc.setStatus(StatusStopped)
			sc.runMu.Unlock()
			return
		}

		sc.setStatus(StatusStarted)
		if sc.Started != nil {
			sc.Started <- sc.current
		}

		sc.lastMu.Lock()
		sc.lastErr = sc.current.Wait()
		sc.lastStatus = exitStatus(sc.lastErr)
		sc.lastMu.Unlock()
		if sc.Status() != StatusStopped {
			sc.setStatus(StatusExited)
		}

		sc.waitMu.Lock()
		if sc.waitCh != nil {
			sc.waitCh <- sc.lastErr
			sc.waitCh = nil
		}
		sc.waitMu.Unlock()

		if sc.lastErr != nil && sc.lastStatus != -1 { // was not SIGKILLED
			sc.runMu.Unlock()
			return
		}
		sc.runMu.Unlock()

		sc.restartMu.Lock()
		if sc.maxRestarts >= 0 && sc.restarts >= sc.maxRestarts {
			sc.restartMu.Unlock()
			return
		}
		sc.restartMu.Unlock()

		if sc.Status() != StatusStopped {
			if err := sc.start(); err != nil {
				panic(err)
			}
			sc.restartMu.Lock()
			sc.restarts++
			sc.restartMu.Unlock()
		}

	}()
	return <-startErr
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
