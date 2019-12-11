package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/manifold/tractor/pkg/data/icons"
	"github.com/manifold/tractor/pkg/logging"
)

type WorkspaceStatus int

const (
	StatusAvailable = iota
	StatusPartially
	StatusUnavailable
)

func (s WorkspaceStatus) Icon() []byte {
	switch int(s) {
	case 0:
		return icons.Available
	case 1:
		return icons.Partially
	default:
		return icons.Unavailable
	}
}

func (s WorkspaceStatus) String() string {
	switch int(s) {
	case 0:
		return "Available"
	case 1:
		return "Partially"
	default:
		return "Unavailable"
	}
}

type Workspace struct {
	Name        string // base name of dir (~/.tractor/workspaces/{name})
	SymlinkPath string // absolute path to symlink file (~/.tractor/workspaces/{name})
	TargetPath  string // absolute path to target of symlink (actual workspace)
	SocketPath  string // absolute path to socket file (~/.tractor/sockets/{name}.sock)
	Status      WorkspaceStatus

	log             logging.Logger
	statusCallbacks []func(*Workspace)
	goBin           string
	consoleBuf      *Buffer
	daemonCmd       *exec.Cmd

	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewWorkspace(a *Agent, name string) *Workspace {
	// JL: this may work differently when symlinks are automatically created
	symlinkPath := filepath.Join(a.WorkspacesPath, name)
	targetPath, err := os.Readlink(symlinkPath)
	if err != nil {
		// JL: 	this is just until the real NewWorkspace API is determined,
		// 		depending on where the symlinking is done. if here, I imagine
		//		NewWorkspace would return (*Workspace, error), but maybe not if
		//		linking happens elsewhere
		panic(err)
	}
	return &Workspace{
		Name:            name,
		SymlinkPath:     symlinkPath,
		TargetPath:      targetPath,
		SocketPath:      filepath.Join(a.WorkspaceSocketsPath, fmt.Sprintf("%s.sock", name)),
		Status:          StatusPartially,
		goBin:           a.GoBin,
		statusCallbacks: make([]func(*Workspace), 0),
		log:             a.Logger,
	}
}

func (w *Workspace) Connect() (io.ReadCloser, error) {
	w.mu.Lock()
	w.log.Info("[workspace]", w.Name, "Connect()")
	if w.consoleBuf != nil {
		w.setStatus(StatusAvailable)
		out := w.consoleBuf.Pipe()
		w.mu.Unlock()

		return out, nil
	}

	err := w.start()
	out := w.consoleBuf.Pipe()
	w.mu.Unlock()
	return out, err
}

// Start starts the workspace daemon. creates the symlink to the path if it does
// not exist, using the path basename as the symlink name
func (w *Workspace) Start() error {
	w.mu.Lock()
	w.log.Info("[workspace]", w.Name, "Start()")

	w.resetPid(StatusPartially)

	err := w.start()
	w.mu.Unlock()
	return err
}

// must run this when the w.mu mutex is locked
func (w *Workspace) start() error {
	buf, err := NewBuffer(1024 * 1024)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.daemonCmd = exec.CommandContext(ctx, w.goBin, "run", "workspace.go",
		"-proto", "unix", "-addr", w.SocketPath)
	w.daemonCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	w.daemonCmd.Dir = w.TargetPath
	w.daemonCmd.Stdout = buf
	w.daemonCmd.Stderr = buf
	w.cancel = cancel

	if err := w.daemonCmd.Start(); err != nil {
		w.setStatus(StatusUnavailable)
		return err
	}

	w.consoleBuf = buf
	w.setStatus(StatusAvailable)

	go func(c *exec.Cmd, ws *Workspace) {
		if err := c.Wait(); err != nil {
			ws.afterWait(c, StatusUnavailable)
			return
		}
		ws.afterWait(c, StatusPartially)
	}(w.daemonCmd, w)

	return nil
}

// Stop stops the workspace daemon, deleting the unix socket file.
func (w *Workspace) Stop() {
	w.mu.Lock()
	w.log.Info("[workspace]", w.Name, "Stop()")
	w.resetPid(StatusPartially)
	w.mu.Unlock()
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

// weird method: resets cmd buffer/pid, sets the menu item status, and returns
// the pid for Close()
// must run when the w.mu mutex is locked.
func (w *Workspace) resetPid(s WorkspaceStatus) {
	if w.cancel != nil {
		w.cancel()
	}
	w.cancel = nil

	if w.daemonCmd != nil {
		w.daemonCmd.Wait()
	}
	w.daemonCmd = nil

	// workplace/init package should clean up its own socket
	os.RemoveAll(w.SocketPath)

	if w.consoleBuf != nil {
		w.consoleBuf.Close()
		w.consoleBuf = nil
	}

	w.setStatus(s)
}

func (w *Workspace) afterWait(c *exec.Cmd, s WorkspaceStatus) {
	w.mu.Lock()
	if c == w.daemonCmd {
		w.resetPid(s)
	}
	w.mu.Unlock()
}

// always run when w.mu mutex is locked
func (w *Workspace) setStatus(s WorkspaceStatus) {
	if w.Status == s {
		return
	}

	w.log.Info("[workspace]", w.Name, "state:", w.Status, "=>", s)
	w.Status = s
	for _, cb := range w.statusCallbacks {
		cb(w)
	}
}
