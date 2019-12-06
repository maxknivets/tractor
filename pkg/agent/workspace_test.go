package agent

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspace(t *testing.T) {
	ag, teardown := setup(t, "test1", "test2", "test3")
	defer teardown()

	t.Run("start/stop", func(t *testing.T) {
		ws := ag.Workspace("test1")
		require.NotNil(t, ws)
		assert.Equal(t, int(StatusPartially), int(ws.Status))

		assert.Nil(t, ws.Start())
		time.Sleep(time.Second)
		assert.Equal(t, int(StatusAvailable), int(ws.Status))

		ws.Stop()
		assert.Equal(t, int(StatusPartially), int(ws.Status))
	})

	t.Run("start/connect/stop", func(t *testing.T) {
		ws := ag.Workspace("test2")
		require.NotNil(t, ws)
		assert.Equal(t, int(StatusPartially), int(ws.Status))

		assert.Nil(t, ws.Start())
		time.Sleep(time.Second)
		assert.Equal(t, int(StatusAvailable), int(ws.Status))

		connCh := readWorkspace(t, ws.Connect)
		time.Sleep(time.Second)

		ws.Stop()
		assert.Equal(t, int(StatusPartially), int(ws.Status))

		connOut := strings.TrimSpace(string(<-connCh))
		assert.True(t, strings.HasPrefix(connOut, "pid "))
	})

	t.Run("connect/stop", func(t *testing.T) {
		ws := ag.Workspace("test3")
		require.NotNil(t, ws)
		assert.Equal(t, int(StatusPartially), int(ws.Status))

		connCh := readWorkspace(t, ws.Connect)
		time.Sleep(time.Second)
		assert.Equal(t, int(StatusAvailable), int(ws.Status))

		ws.Stop()
		assert.Equal(t, int(StatusPartially), int(ws.Status))

		connOut := strings.TrimSpace(string(<-connCh))
		assert.True(t, strings.HasPrefix(connOut, "pid "))
	})

	t.Run("erroring workspace", func(t *testing.T) {
		ws := ag.Workspace("err")
		require.NotNil(t, ws)
		assert.Equal(t, int(StatusPartially), int(ws.Status))

		startCh := readWorkspace(t, ws.Connect)
		time.Sleep(time.Second)
		assert.Equal(t, int(StatusUnavailable), int(ws.Status))

		startOut := strings.TrimSpace(string(<-startCh))
		assert.True(t, strings.HasPrefix(startOut, "boomtown "))
	})
}

func readWorkspace(t *testing.T, wsFunc func() (io.ReadCloser, error)) chan []byte {
	ch := make(chan []byte)
	go func() {
		r, err := wsFunc()
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
