package subcmd

// func TestSubcmdStartWait(t *testing.T) {
// 	cmd := New("echo")
// 	cmd.MaxRestarts = 0
// 	assert.Equal(t, StatusStopped, cmd.Status())

// 	assert.Nil(t, cmd.Start())
// 	assert.Equal(t, StatusStarting, cmd.Status())

// 	assert.Nil(t, cmd.Wait())
// 	assert.Equal(t, StatusExited, cmd.Status())
// }

// func TestSubcmdStartStopFast(t *testing.T) {
// 	cmd := New("echo")
// 	cmd.MaxRestarts = 0
// 	assert.Equal(t, StatusStopped, cmd.Status())

// 	assert.Nil(t, cmd.Start())
// 	assert.Equal(t, StatusStarting, cmd.Status())

// 	assert.Nil(t, cmd.Stop())
// 	assert.Equal(t, StatusStopped, cmd.Status())
// }
