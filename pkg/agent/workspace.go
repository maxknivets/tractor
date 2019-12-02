package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/manifold/tractor/pkg/agent/icons"
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
	Socket      string // absolute path to socket file (~/.tractor/sockets/{name}.sock)
	Status      WorkspaceStatus
	bin         string
	buf         *Buffer
	callbacks   []func(*Workspace)
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	mu          sync.Mutex
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
		Name:        name,
		SymlinkPath: symlinkPath,
		TargetPath:  targetPath,
		Socket:      filepath.Join(a.SocketsPath, fmt.Sprintf("%s.sock", name)),
		Status:      StatusPartially,
		bin:         a.Bin,
		callbacks:   make([]func(*Workspace), 0),
	}
}

func (w *Workspace) Connect() (io.ReadCloser, error) {
	w.mu.Lock()
	log.Println("[workspace]", w.Name, "Connect()")
	if w.buf != nil {
		w.setStatus(StatusAvailable)
		out := w.buf.Pipe()
		w.mu.Unlock()

		return out, nil
	}

	out, err := w.start()
	w.mu.Unlock()
	return out, err
}

// Start starts the workspace daemon. creates the symlink to the path if it does
// not exist, using the path basename as the symlink name
func (w *Workspace) Start() (io.ReadCloser, error) {
	w.mu.Lock()
	log.Println("[workspace]", w.Name, "Start()")

	w.resetPid(StatusPartially)

	out, err := w.start()
	w.mu.Unlock()
	return out, err
}

// must run this when the w.mu mutex is locked
func (w *Workspace) start() (io.ReadCloser, error) {
	buf, err := NewBuffer(1024 * 1024)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cmd = exec.CommandContext(ctx, w.bin, "run", "workspace.go",
		"-proto", "unix", "-addr", w.Socket)
	w.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	w.cmd.Dir = w.TargetPath
	w.cmd.Stdout = buf
	w.cmd.Stderr = buf
	w.cancel = cancel

	if err := w.cmd.Start(); err != nil {
		w.setStatus(StatusUnavailable)
		return nil, err
	}

	w.buf = buf
	w.setStatus(StatusAvailable)

	go func(c *exec.Cmd, ws *Workspace) {
		if err := c.Wait(); err != nil {
			ws.afterWait(c, StatusUnavailable)
			return
		}
		ws.afterWait(c, StatusPartially)
	}(w.cmd, w)

	return buf.Pipe(), nil
}

// Stop stops the workspace daemon, deleting the unix socket file.
func (w *Workspace) Stop() {
	w.mu.Lock()
	log.Println("[workspace]", w.Name, "Stop()")
	w.resetPid(StatusPartially)
	w.mu.Unlock()
}

func (w *Workspace) OnStatusChange(cb func(*Workspace)) {
	cb(w)
	w.mu.Lock()
	w.callbacks = append(w.callbacks, cb)
	w.mu.Unlock()
}

func (w *Workspace) BufferStatus() (int, int64) {
	return w.buf.Status()
}

// weird method: resets cmd buffer/pid, sets the menu item status, and returns
// the pid for Close()
// must run when the w.mu mutex is locked.
func (w *Workspace) resetPid(s WorkspaceStatus) {
	if w.cancel != nil {
		w.cancel()
	}
	w.cancel = nil

	if w.cmd != nil {
		w.cmd.Wait()
	}
	w.cmd = nil

	// workplace/init package should clean up its own socket
	os.RemoveAll(w.Socket)

	if w.buf != nil {
		w.buf.Close()
		w.buf = nil
	}

	w.setStatus(s)
}

func (w *Workspace) afterWait(c *exec.Cmd, s WorkspaceStatus) {
	w.mu.Lock()
	if c == w.cmd {
		w.resetPid(s)
	}
	w.mu.Unlock()
}

// always run when w.mu mutex is locked
func (w *Workspace) setStatus(s WorkspaceStatus) {
	if w.Status == s {
		return
	}

	log.Println("[workspace]", w.Name, "state:", w.Status, "=>", s)
	w.Status = s
	for _, cb := range w.callbacks {
		cb(w)
	}
}
