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
	pkgpath    = filepath.Dir(b)
)

func TestAgent(t *testing.T) {
	ag, teardown := setup(t)
	defer teardown()

	t.Logf("tmp=%q", os.TempDir())
	t.Logf("path=%q", ag.Path)

	t.Run("paths", func(t *testing.T) {
		assert.True(t, strings.HasPrefix(ag.Path, os.TempDir()))
		assert.Equal(t, filepath.Join(ag.Path, "agent.sock"), ag.SocketPath)
		assert.Equal(t, filepath.Join(ag.Path, "workspaces"), ag.WorkspacesPath)
		assert.Equal(t, filepath.Join(ag.Path, "sockets"), ag.WorkspaceSocketsPath)
	})

	t.Run("finds workspaces", func(t *testing.T) {
		wss, err := ag.Workspaces()
		assert.Nil(t, err)
		require.Equal(t, 2, len(wss))
		assert.Equal(t, "err", wss[0].Name)
		assert.Equal(t, filepath.Join(ag.WorkspacesPath, "err"), wss[0].SymlinkPath)
		assert.Equal(t, "test", wss[1].Name)
		assert.Equal(t, filepath.Join(ag.WorkspacesPath, "test"), wss[1].SymlinkPath)
	})

	t.Run("finds existing workspace", func(t *testing.T) {
		require.NotNil(t, ag.Workspace("test"))
	})

	t.Run("attempts find missing workspace", func(t *testing.T) {
		require.Nil(t, ag.Workspace("nope"))
	})
}

func setup(t *testing.T, extradirs ...string) (*Agent, func()) {
	// use tempdir in place of ~/.tractor
	dirname, err := ioutil.TempDir("", "tractor-pkg-agent-"+t.Name())
	assert.Nil(t, err)

	ag := newAgent(t, dirname)
	err = os.Symlink(filepath.Join(pkgpath, "errworkspace"), filepath.Join(ag.WorkspacesPath, "err"))
	assert.Nil(t, err)

	wspath := filepath.Join(pkgpath, "testworkspace")
	for _, n := range append(extradirs, "test") {
		err = os.Symlink(wspath, filepath.Join(ag.WorkspacesPath, n))
		assert.Nil(t, err)
	}

	return ag, func() {
		ag.Shutdown()
		os.RemoveAll(dirname)
	}
}

func newAgent(t *testing.T, path string) *Agent {
	ag, err := Open(path)
	assert.Nil(t, err)
	return ag
}
