package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/manifold/tractor/pkg/logging"
)

// Agent manages multiple workspaces in a directory (default: ~/.tractor).
type Agent struct {
	Path                 string // ~/.tractor
	SocketPath           string // ~/.tractor/agent.sock
	WorkspacesPath       string // ~/.tractor/workspaces
	WorkspaceSocketsPath string // ~/.tractor/sockets
	GoBin                string

	Logger logging.Logger

	workspaces map[string]*Workspace
	mu         sync.RWMutex
}

// Open returns a new agent for the given path. If the given path is empty, a
// default of ~/.tractor will be used.
func Open(path string) (*Agent, error) {
	bin, err := exec.LookPath("go")
	if err != nil {
		return nil, err
	}

	a := &Agent{
		Path:       path,
		GoBin:      bin,
		workspaces: make(map[string]*Workspace),
	}

	if len(a.Path) == 0 {
		p, err := defaultPath()
		if err != nil {
			return nil, err
		}
		a.Path = p
	}

	a.SocketPath = filepath.Join(a.Path, "agent.sock")
	a.WorkspacesPath = filepath.Join(a.Path, "workspaces")
	a.WorkspaceSocketsPath = filepath.Join(a.Path, "sockets")

	os.MkdirAll(a.WorkspacesPath, 0700)
	os.MkdirAll(a.WorkspaceSocketsPath, 0700)

	return a, nil
}

// Workspace returns a Workspace for the given path. The path must match
// either:
//   * the workspace symlink's basename in the agent's WorkspacesPath.
//   * the full path to the target of a workspace symlink in WorkspacesPath.
//   * full path to the workspace anywhere else. it will be symlinked to
//     the Workspaces path using the basename of the full path.
func (a *Agent) Workspace(path string) *Workspace {
	// check to see if the workspace is cached
	// cached=workspace is running through an agent QRPC call, or showing the
	// workspace in the systray.
	a.mu.RLock()
	ws := a.workspaces[path]
	a.mu.RUnlock()
	if ws != nil {
		return ws
	}

	// now look for a symlink in ~/.tractor/workspaces
	wss, _ := a.Workspaces()
	for _, ws := range wss {
		if ws.Name == path || ws.TargetPath == path {
			return ws
		}
	}

	// if full path is a dir with workspace.go, symlink it
	basename, err := a.symlinkWorkspace(path)
	if err != nil {
		return nil
	}

	return a.Workspace(basename)
}

func (a *Agent) symlinkWorkspace(path string) (string, error) {
	fi, err := os.Lstat(filepath.Join(path, "workspace.go"))
	if err != nil {
		return "", err
	}

	if fi.IsDir() {
		return "", nil
	}

	basepath := filepath.Base(path)
	base := basepath
	i := 1
	for {
		err = os.Symlink(path, filepath.Join(a.WorkspacesPath, base))
		if err != nil && !os.IsExist(err) {
			return base, err
		}

		if err == nil {
			return base, nil
		}

		i++
		base = fmt.Sprintf("%s-%d", basepath, i)
	}
}

// Workspaces returns the workspaces under this agent's WorkspacesPath.
func (a *Agent) Workspaces() ([]*Workspace, error) {
	entries, err := ioutil.ReadDir(a.WorkspacesPath)
	if err != nil {
		return nil, err
	}

	workspaces := make([]*Workspace, 0, len(entries))
	a.mu.Lock()
	for _, entry := range entries {
		if !a.isWorkspaceDir(entry) {
			continue
		}

		n := entry.Name()
		ws := a.workspaces[n]
		if ws == nil {
			ws = NewWorkspace(a, n)
			a.workspaces[n] = ws
		}
		workspaces = append(workspaces, ws)
	}
	a.mu.Unlock()
	return workspaces, nil
}

// Shutdown shuts all workspaces down and cleans up socket files.
func (a *Agent) Shutdown() {
	a.Logger.Info("[server] shutting down")
	os.RemoveAll(a.SocketPath)
	for _, ws := range a.workspaces {
		ws.Stop()
	}
}

func (a *Agent) Watch(ctx context.Context, ch chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		a.Logger.Info("unable to create watcher:", err)
		return
	}
	watcher.Add(a.WorkspacesPath)
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

			ch <- struct{}{}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			a.Logger.Debug("watcher error:", err)
		}
	}
}

func (a *Agent) isWorkspaceDir(fi os.FileInfo) bool {
	if fi.IsDir() {
		return true
	}

	path := filepath.Join(a.WorkspacesPath, fi.Name())
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		a.Logger.Error(err)
		return false
	}

	if resolved == path {
		return false
	}

	rfi, err := os.Lstat(resolved)
	if err != nil {
		a.Logger.Error(err)
		return false
	}

	return rfi.IsDir()
}

func defaultPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(usr.HomeDir, ".tractor"), nil
}
