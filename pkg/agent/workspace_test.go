package agent

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspace(t *testing.T) {
	ag, teardown := setup(t, "test1", "test2", "test3")
	defer teardown()

	t.Run("stop/start", func(t *testing.T) {
		status, ws := setupWorkspace(t, ag, "test1")
		assert.Equal(t, StatusAvailable, <-status)

		assert.Nil(t, ws.Stop())
		assert.Equal(t, StatusUnavailable, <-status)

		assert.Nil(t, ws.Start())
		assert.Equal(t, StatusAvailable, <-status)
	})

	t.Run("connect/stop", func(t *testing.T) {
		status, ws := setupWorkspace(t, ag, "test2")
		assert.Equal(t, StatusAvailable, <-status)

		out, err := ws.Connect()
		assert.NoError(t, err)
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			assert.True(t, strings.HasPrefix(scanner.Text(), "pid "))
			break
		}

		assert.NoError(t, ws.Stop())
		assert.Equal(t, StatusUnavailable, <-status)
	})

	// t.Run("connect/stop", func(t *testing.T) {
	// 	ws := ag.Workspace("test3")
	// 	require.NotNil(t, ws)
	// 	assert.Equal(t, StatusAvailable, ws.Status)

	// 	connCh := readWorkspace(t, ws.Connect)
	// 	time.Sleep(time.Second)
	// 	assert.Equal(t, StatusAvailable, ws.Status)

	// 	ws.Stop()
	// 	assert.Equal(t, StatusUnavailable, ws.Status)

	// 	connOut := strings.TrimSpace(string(<-connCh))
	// 	assert.True(t, strings.HasPrefix(connOut, "pid "))
	// })

	t.Run("erroring workspace", func(t *testing.T) {
		status, ws := setupWorkspace(t, ag, "err")
		assert.Equal(t, StatusAvailable, <-status)

		out, err := ws.Connect()
		assert.NoError(t, err)
		b, err := ioutil.ReadAll(out)
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(b), "boomtown "))

		assert.Equal(t, 1, ws.daemon.ExitStatus())
		assert.Equal(t, StatusUnavailable, <-status)
	})
}

func setupWorkspace(t *testing.T, ag *Agent, name string) (chan WorkspaceStatus, *Workspace) {
	status := make(chan WorkspaceStatus, 3)
	ws := ag.Workspace(name)
	require.NotNil(t, ws)
	//ws.SetDaemonCmd("cat")
	ws.Observe(func(_ *Workspace, newStatus WorkspaceStatus) {
		status <- newStatus
	})
	require.NoError(t, ws.StartDaemon())
	return status, ws
}

func workspaceConnect(t *testing.T, ws *Workspace) chan []byte {
	ch := make(chan []byte)
	go func() {
		r, err := ws.Connect()
		if err != nil {
			t.Error(err)
			return
		}

		out := &bytes.Buffer{}
		by := make([]byte, 10)
		for {
			n, err := r.Read(by)
			if err != nil {
				if err != io.EOF {
					t.Error(err)
				}
				break
			}
			out.Write(by[0:n])
		}
		ch <- out.Bytes()
	}()
	return ch
}
