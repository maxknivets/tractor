package agent

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/manifold/tractor/pkg/agent/console"
	"github.com/manifold/tractor/pkg/data/icons"
	"github.com/manifold/tractor/pkg/misc/buffer"
	"github.com/manifold/tractor/pkg/misc/logging"
	"github.com/manifold/tractor/pkg/misc/subcmd"
)

type WorkspaceStatus string

const (
	StatusAvailable   WorkspaceStatus = "Available"
	StatusPartially   WorkspaceStatus = "Partially"
	StatusUnavailable WorkspaceStatus = "Unavailable"
)

func (s WorkspaceStatus) Icon() []byte {
	switch s {
	case StatusAvailable:
		return icons.Available
	case StatusPartially:
		return icons.Partially
	default:
		return icons.Unavailable
	}
}

func (s WorkspaceStatus) String() string {
	return string(s)
}

type Workspace struct {
	Name        string // base name of dir (~/.tractor/workspaces/{name})
	SymlinkPath string // absolute path to symlink file (~/.tractor/workspaces/{name})
	TargetPath  string // absolute path to target of symlink (actual workspace)
	SocketPath  string // absolute path to socket file (~/.tractor/sockets/{name}.sock)
	Status      WorkspaceStatus

	log             logging.Logger
	consolePipe     io.WriteCloser
	statusCallbacks []func(*Workspace)
	goBin           string
	consoleBuf      *buffer.Buffer
	daemon          *subcmd.Subcmd

	starting sync.Mutex
	mu       sync.Mutex
}

func InitWorkspace(a *Agent, name string) (*Workspace, error) {
	symlinkPath := filepath.Join(a.WorkspacesPath, name)
	targetPath, err := os.Readlink(symlinkPath)
	if err != nil {
		return nil, err
	}
	var consolePipe io.WriteCloser
	if svc, ok := a.Logger.(*console.Service); ok {
		consolePipe = svc.NewPipe(name)
	}
	ws := &Workspace{
		Name:            name,
		SymlinkPath:     symlinkPath,
		TargetPath:      targetPath,
		SocketPath:      filepath.Join(a.WorkspaceSocketsPath, fmt.Sprintf("%s.sock", name)),
		Status:          StatusPartially,
		goBin:           a.GoBin,
		statusCallbacks: make([]func(*Workspace), 0),
		log:             a.Logger,
		consolePipe:     consolePipe,
	}
	ws.consoleBuf, err = buffer.NewBuffer(1024 * 1024)
	if err != nil {
		return nil, err
	}
	if err = ws.startDaemon(); err != nil {
		return nil, err
	}
	return ws, nil
}

func (w *Workspace) startDaemon() error {
	w.daemon = subcmd.New(w.goBin, "run", "workspace.go",
		"-proto", "unix", "-addr", w.SocketPath)
	w.daemon.Setup = func(cmd *exec.Cmd) error {
		w.consoleBuf.Reset()

		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Dir = w.TargetPath
		if w.consolePipe != nil {
			cmd.Stdout = io.MultiWriter(w.consoleBuf, w.consolePipe)
			cmd.Stderr = io.MultiWriter(w.consoleBuf, w.consolePipe)
		} else {
			cmd.Stdout = w.consoleBuf
			cmd.Stderr = w.consoleBuf
		}

		return nil
	}

	if err := w.daemon.Start(); err != nil {
		w.setStatus(StatusUnavailable)
		return err
	}

	w.daemon.OnStatusChange(func(cmd *subcmd.Subcmd) {
		switch cmd.Status {
		case subcmd.StatusStarted:
			w.setStatus(StatusAvailable)
		case subcmd.StatusExited:
			w.cleanup()
			if cmd.Error() != nil {
				info(w.log, cmd.Error())
				w.setStatus(StatusUnavailable)
			} else {
				w.setStatus(StatusPartially)
			}
		case subcmd.StatusStopped:
			w.cleanup()
			w.setStatus(StatusUnavailable)
		}
	})

	return nil
}

func (w *Workspace) cleanup() {
	// workplace/init package should clean up its own socket
	os.RemoveAll(w.SocketPath)
	w.consoleBuf.Close()
}

func (w *Workspace) Connect() (io.ReadCloser, error) {
	info(w.log, "[workspace]", w.Name, "Connect()")
	var err error
	if !subcmd.Running(w.daemon) {
		err = w.daemon.Start()
	}
	out := w.consoleBuf.Pipe()
	return out, err
}

// Start starts the workspace daemon. creates the symlink to the path if it does
// not exist, using the path basename as the symlink name
func (w *Workspace) Start() error {
	info(w.log, "[workspace]", w.Name, "Start()")
	return w.daemon.Restart()
}

// Stop stops the workspace daemon, deleting the unix socket file.
func (w *Workspace) Stop() error {
	info(w.log, "[workspace]", w.Name, "Stop()")
	return w.daemon.Stop()
}

func (w *Workspace) OnStatusChange(cb func(*Workspace)) {
	cb(w)
	w.mu.Lock()
	w.statusCallbacks = append(w.statusCallbacks, cb)
	w.mu.Unlock()
}

func (w *Workspace) BufferStatus() (int, int64) {
	return w.consoleBuf.Status()
}

// always run when w.mu mutex is locked
func (w *Workspace) setStatus(s WorkspaceStatus) {
	if w.Status == s {
		return
	}
	info(w.log, "[workspace]", w.Name, "state:", w.Status, "=>", s)

	w.mu.Lock()
	defer w.mu.Unlock()
	w.Status = s
	for _, cb := range w.statusCallbacks {
		cb(w)
	}
}
