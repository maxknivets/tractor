package agent

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
)

type Agent struct {
	Path                 string // ~/.tractor
	SocketPath           string // ~/.tractor/agent.sock
	WorkspacesPath       string // ~/.tractor/workspaces
	WorkspaceSocketsPath string // ~/.tractor/sockets
	GoBin                string

	workspaces map[string]*Workspace
	mu         sync.RWMutex
}

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

func (a *Agent) Workspace(path string) *Workspace {
	a.mu.RLock()
	ws := a.workspaces[path]
	a.mu.RUnlock()
	if ws != nil {
		return ws
	}

	wss, _ := a.Workspaces()
	for _, ws := range wss {
		if ws.Name == path || ws.TargetPath == path {
			return ws
		}
	}

	return nil
}

func (a *Agent) Shutdown() {
	log.Println("[server] shutting down")
	os.RemoveAll(a.SocketPath)
	for _, ws := range a.workspaces {
		ws.Stop()
	}
}

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

func (a *Agent) isWorkspaceDir(fi os.FileInfo) bool {
	if fi.IsDir() {
		return true
	}

	path := filepath.Join(a.WorkspacesPath, fi.Name())
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		log.Println(err)
		return false
	}

	if resolved == path {
		return false
	}

	rfi, err := os.Lstat(resolved)
	if err != nil {
		log.Println(err)
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
