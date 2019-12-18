package subcmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCmd(lrp bool, prog string, args ...string) (chan Status, *Subcmd) {
	status := make(chan Status, 3)
	cmd := New(prog, args...)
	if lrp {
		cmd.Setup = func(cmd *exec.Cmd) error {
			cmd.StdinPipe()
			return nil
		}
	} else {
		cmd.SetMaxRestarts(0)
	}
	//cmd.Log = std.NewLogger("", os.Stdout)
	cmd.Observe(func(sc *Subcmd, newStatus Status) {
		status <- newStatus
	})
	return status, cmd
}

func TestStartWait(t *testing.T) {
	status, cmd := setupCmd(false, "echo")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.Nil(t, cmd.Wait())
	assert.Equal(t, StatusExited, <-status)
}

func TestWaitNonZero(t *testing.T) {
	status, cmd := setupCmd(false, "sh", "-c", "exit 2")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.NotNil(t, cmd.Wait())
	assert.Equal(t, StatusExited, <-status)
	assert.Equal(t, 2, cmd.ExitStatus())
}

func TestStartStopExited(t *testing.T) {
	status, cmd := setupCmd(false, "echo")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)
	assert.Equal(t, StatusExited, <-status)

	err := cmd.Stop()
	if assert.Error(t, err) {
		assert.Equal(t, ErrNotRunning, err)
	}
}

func TestStartStopFast(t *testing.T) {
	status, cmd := setupCmd(false, "echo")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)

	assert.Nil(t, cmd.Stop())
	assert.Equal(t, StatusStarted, <-status)
	assert.Equal(t, StatusStopped, <-status)
}

func TestLRPNonZero(t *testing.T) {
	status, cmd := setupCmd(true, "sh", "-c", "exit 1")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)
	assert.Equal(t, StatusExited, <-status)
	assert.Equal(t, 1, cmd.ExitStatus())
}

func TestLRPStartStop(t *testing.T) {
	status, cmd := setupCmd(true, "cat")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.True(t, Running(cmd))

	assert.Nil(t, cmd.Stop())
	assert.Equal(t, StatusStopped, <-status)
}

func TestLRPRestartStop(t *testing.T) {
	status, cmd := setupCmd(true, "cat")
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Restart())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.True(t, Running(cmd))

	assert.Nil(t, cmd.Stop())
	assert.Equal(t, StatusStopped, <-status)
}

func TestLRPStartRestartStop(t *testing.T) {
	status, cmd := setupCmd(true, "cat")
	cmd.SetMaxRestarts(1)
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.Nil(t, cmd.Restart())
	assert.Equal(t, StatusExited, <-status)
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.Nil(t, cmd.Stop())
	assert.Equal(t, StatusStopped, <-status)
}

func TestLRPMaxRestarts(t *testing.T) {
	status, cmd := setupCmd(true, "cat")
	cmd.SetMaxRestarts(1)
	require.Equal(t, StatusStopped, cmd.Status())

	assert.Nil(t, cmd.Start())
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.Nil(t, cmd.terminate())
	assert.Equal(t, StatusExited, <-status)
	assert.Equal(t, StatusStarting, <-status)
	assert.Equal(t, StatusStarted, <-status)

	assert.Nil(t, cmd.terminate())
	assert.Equal(t, StatusExited, <-status)
	assert.False(t, Running(cmd))
}
