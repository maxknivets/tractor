package buffer_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/manifold/tractor/pkg/misc/buffer"
	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	buf, err := buffer.NewBuffer(3)
	assert.Nil(t, err)

	numPipes, sizeSeen := buf.Status()
	assert.Equal(t, 0, numPipes)
	assert.Equal(t, int64(0), sizeSeen)

	pr1 := buf.Pipe()
	ch1 := readAll(t, pr1)

	numPipes, sizeSeen = buf.Status()
	assert.Equal(t, 1, numPipes)
	assert.Equal(t, int64(0), sizeSeen)

	buf.Write([]byte("ab"))

	numPipes, sizeSeen = buf.Status()
	assert.Equal(t, 1, numPipes)
	assert.Equal(t, int64(2), sizeSeen)

	pr2 := buf.Pipe()
	ch2 := readAll(t, pr2)

	numPipes, sizeSeen = buf.Status()
	assert.Equal(t, 2, numPipes)
	assert.Equal(t, int64(2), sizeSeen)

	buf.Write([]byte("cd"))

	pr3 := buf.Pipe()
	ch3 := readAll(t, pr3)

	buf.Write([]byte("ef"))

	numPipes, sizeSeen = buf.Status()
	assert.Equal(t, 3, numPipes)
	assert.Equal(t, int64(6), sizeSeen)

	buf.Close()
	assert.Equal(t, "abcdef", string(<-ch1))
	assert.Equal(t, "abcdef", string(<-ch2))
	assert.Equal(t, "bcdef", string(<-ch3))
}

func readAll(t *testing.T, r io.Reader) chan []byte {
	ch := make(chan []byte)
	go func() {
		out := &bytes.Buffer{}
		by := make([]byte, 10)
		for {
			n, err := r.Read(by)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Error(err)
			}
			out.Write(by[0:n])
		}
		ch <- out.Bytes()
	}()
	return ch
}
