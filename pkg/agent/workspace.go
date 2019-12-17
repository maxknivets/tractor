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
	"time"

	"github.com/fsnotify/fsnotify"
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

	log             logging.Logger
	status          WorkspaceStatus
	consolePipe     io.WriteCloser
	statusCallbacks []func(*Workspace)
	goBin           string
	consoleBuf      *buffer.Buffer
	daemon          *subcmd.Subcmd

	starting sync.Mutex
	statMu   sync.Mutex
	cbMu     sync.Mutex
}

func InitWorkspace(a *Agent, name string) (*Workspace, error) {
	symlinkPath := filepath.Join(a.WorkspacesPath, name)
	targetPath, err := os.Readlink(symlinkPath)
	if err != nil {
		return nil, err
	}
	var consolePipe io.WriteCloser
	if svc, ok := a.Logger.(*console.Service); ok && svc != nil {
		consolePipe = svc.NewPipe(name)
	}
	ws := &Workspace{
		Name:            name,
		SymlinkPath:     symlinkPath,
		TargetPath:      targetPath,
		SocketPath:      filepath.Join(a.WorkspaceSocketsPath, fmt.Sprintf("%s.sock", name)),
		status:          StatusPartially,
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

func (w *Workspace) Status() WorkspaceStatus {
	w.statMu.Lock()
	defer w.statMu.Unlock()
	return w.status
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
		switch cmd.Status() {
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

func (w *Workspace) Serve(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		info(w.log, "unable to create watcher:", err)
		return
	}
	for _, path := range collectDirs(w.TargetPath, []string{"node_modules", ".git"}) {
		err = watcher.Add(path)
		if err != nil {
			info(w.log, "unable to watch path:", path, err)
			return
		}
	}
	debounce := Debounce(20 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			watcher.Close()
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			dirCreated := false
			if event.Op&fsnotify.Create == fsnotify.Create {
				fi, err := os.Stat(event.Name)
				if err != nil {
					info(w.log, "watcher:", err)
				}
				if fi.IsDir() {
					watcher.Add(event.Name)
					dirCreated = true
				}
			}

			if filepath.Ext(event.Name) != ".go" && !dirCreated {
				continue
			}

			debounce(func() {
				info(w.log, "reloading workspace:", w.Name)
				w.daemon.Restart()
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logErr(w.log, "watcher error:", err)
		}
	}
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
	w.cbMu.Lock()
	w.statusCallbacks = append(w.statusCallbacks, cb)
	w.cbMu.Unlock()
}

func (w *Workspace) BufferStatus() (int, int64) {
	return w.consoleBuf.Status()
}

func (w *Workspace) setStatus(s WorkspaceStatus) {
	w.statMu.Lock()
	if w.status == s {
		w.statMu.Unlock()
		return
	}
	info(w.log, "[workspace]", w.Name, "state:", w.status, "=>", s)

	w.status = s
	w.statMu.Unlock()
	w.cbMu.Lock()
	for _, cb := range w.statusCallbacks {
		cb(w)
	}
	w.cbMu.Unlock()
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
