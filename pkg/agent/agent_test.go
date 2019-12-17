package agent

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_, b, _, _ = runtime.Caller(0)
	pkgpath    = filepath.Join(filepath.Dir(b), "testutil")
)

func TestAgent(t *testing.T) {
	ag, teardown := setup(t)
	defer teardown()

	t.Run("paths", func(t *testing.T) {
		assert.True(t, strings.HasPrefix(ag.Path, os.TempDir()))
		assert.Equal(t, filepath.Join(ag.Path, "agent.sock"), ag.SocketPath)
		assert.Equal(t, filepath.Join(ag.Path, "workspaces"), ag.WorkspacesPath)
		assert.Equal(t, filepath.Join(ag.Path, "sockets"), ag.WorkspaceSocketsPath)
	})

	t.Run("finds workspaces", func(t *testing.T) {
		wss, err := ag.Workspaces()
		assert.Nil(t, err)

		expected := []string{"err", "test"}
		assert.Equal(t, expected, wsNames(wss))
		for _, ws := range wss {
			assert.Equal(t, filepath.Join(ag.WorkspacesPath, ws.Name), ws.SymlinkPath)
		}
	})

	t.Run("finds existing workspace by symlink basename", func(t *testing.T) {
		require.NotNil(t, ag.Workspace("test"))
	})

	t.Run("attempts to find workspace with missing symlink basename", func(t *testing.T) {
		require.Nil(t, ag.Workspace("nope"))
	})
}

// testing workspaces opened by full path
func TestAgentWorkspaces(t *testing.T) {
	errWsPath := filepath.Join(pkgpath, "errworkspace")
	testWsPath := filepath.Join(pkgpath, "testworkspace")
	ag, teardown := setupAgent(t, func(ag *Agent) {
		err := os.Symlink(errWsPath, filepath.Join(ag.WorkspacesPath, "err"))
		require.Nil(t, err)
	})
	defer teardown()

	t.Run("access symlinked workspace with fullpath", func(t *testing.T) {
		assert.Equal(t, []string{"err"}, agentWsNames(ag))

		existing := ag.Workspace("err")
		require.NotNil(t, existing)

		actual := ag.Workspace(errWsPath)
		assert.Equal(t, existing, actual)

		assert.Equal(t, []string{"err"}, agentWsNames(ag))
	})

	t.Run("access non-symlinked workspace with fullpath", func(t *testing.T) {
		// replace ./tractor/workspaces/testworkspace with existing symlink to
		//   errworkspace
		assert.Equal(t, []string{"err"}, agentWsNames(ag))
		require.Nil(t, os.Symlink(errWsPath, filepath.Join(ag.WorkspacesPath, "testworkspace")))
		assert.Equal(t, []string{"err", "testworkspace"}, agentWsNames(ag))
		require.Nil(t, os.Symlink(errWsPath, filepath.Join(ag.WorkspacesPath, "testworkspace-2")))
		assert.Equal(t, []string{"err", "testworkspace", "testworkspace-2"}, agentWsNames(ag))

		actual := ag.Workspace(testWsPath)
		require.NotNil(t, actual)
		assert.Equal(t, "testworkspace-3", actual.Name)
		assert.Equal(t, []string{"err", "testworkspace", "testworkspace-2", "testworkspace-3"}, agentWsNames(ag))
	})
}

func agentWsNames(ag *Agent) []string {
	wss, _ := ag.Workspaces()
	return wsNames(wss)
}

func wsNames(wss []*Workspace) []string {
	names := make([]string, len(wss))
	for i, ws := range wss {
		names[i] = ws.Name
	}
	return names
}

func setup(t *testing.T, extradirs ...string) (*Agent, func()) {
	return setupAgent(t, func(ag *Agent) {
		err := os.Symlink(filepath.Join(pkgpath, "errworkspace"), filepath.Join(ag.WorkspacesPath, "err"))
		require.Nil(t, err)

		wspath := filepath.Join(pkgpath, "testworkspace")
		for _, n := range append(extradirs, "test") {
			err = os.Symlink(wspath, filepath.Join(ag.WorkspacesPath, n))
			require.Nil(t, err)
		}
	})
}

func setupAgent(t *testing.T, setupFn func(*Agent)) (*Agent, func()) {
	// use tempdir in place of ~/.tractor
	dirname, err := ioutil.TempDir("", "tractor-pkg-agent-"+t.Name())
	assert.Nil(t, err)

	ag := newAgent(t, dirname)

	t.Logf("tmp=%q", dirname)
	t.Logf("path=%q", ag.Path)

	if setupFn != nil {
		setupFn(ag)
	}

	return ag, func() {
		ag.Shutdown()
		os.RemoveAll(dirname)
	}
}

func newAgent(t *testing.T, path string) *Agent {
	ag, err := Open(path, nil, false)
	assert.Nil(t, err)
	return ag
}
