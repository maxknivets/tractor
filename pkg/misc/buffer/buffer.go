package buffer

import (
	"bytes"
	"io"
	"sync"

	"github.com/armon/circbuf"
)

// Buffer captures stdout/stderr for workspace processes. All writes are
// forwarded to a circular buffer, as well as any attached pipe writers. The
// data in the circular buffer is used to preload new connections with some
// recent history.
type Buffer struct {
	buf    *circbuf.Buffer
	pipes  map[*pipeReader]*io.PipeWriter
	muBuf  sync.Mutex   // wraps buf
	muPipe sync.RWMutex // wraps pipes
}

// NewBuffer returns a new Buffer with the given size.
func NewBuffer(size int64) (*Buffer, error) {
	cbuf, err := circbuf.NewBuffer(size)
	if err != nil {
		return nil, err
	}
	return &Buffer{
		buf:   cbuf,
		pipes: make(map[*pipeReader]*io.PipeWriter),
	}, nil
}

func (b *Buffer) Reset() {
	b.muBuf.Lock()
	b.buf.Reset()
	b.muBuf.Unlock()
}

// Status returns how many pipes are currently attached, and how much data has
// passed through this buffer since inception.
func (b *Buffer) Status() (int, int64) {
	if b == nil {
		return 0, 0
	}

	b.muPipe.RLock()
	n := len(b.pipes)
	b.muPipe.RUnlock()

	b.muBuf.Lock()
	tw := b.buf.TotalWritten()
	b.muBuf.Unlock()
	return n, tw
}

// Write writes to the circular buffer and any attached pipe writers.
func (b *Buffer) Write(by []byte) (int, error) {
	b.muBuf.Lock()
	n, err := b.buf.Write(by)
	b.muBuf.Unlock()
	if err != nil {
		return n, err
	}

	b.muPipe.RLock()
	defer b.muPipe.RUnlock()
	for _, pw := range b.pipes {
		pn, err := pw.Write(by)
		if err != nil {
			return n, err
		}
		if pn != n {
			return n, io.ErrShortWrite
		}
	}

	return n, err
}

// Pipe returns a new pipe reader that reads data from this buffer. Closing the
// pipe reader will release it from the buffer too.
func (b *Buffer) Pipe() io.ReadCloser {
	pr, pw := io.Pipe()

	b.muPipe.Lock()
	b.muBuf.Lock()
	by := b.buf.Bytes()
	b.muBuf.Unlock()

	cp := append(by[:0:0], by...)
	pr2 := &pipeReader{
		Reader: io.MultiReader(bytes.NewBuffer(cp), pr),
		pr:     pr,
		buf:    b,
	}

	b.pipes[pr2] = pw
	b.muPipe.Unlock()
	return pr2
}

// Release removes the given pipe reader, and its attached pipe writer from this
// buffer.
func (b *Buffer) Release(r io.ReadCloser) {
	if pr, ok := r.(*pipeReader); ok {
		b.muPipe.Lock()
		delete(b.pipes, pr)
		b.muPipe.Unlock()
	}
}

// Close closes and releases all attached pipes.
func (b *Buffer) Close() error {
	b.muPipe.Lock()
	for pr := range b.pipes {
		pr.pr.Close()
		delete(b.pipes, pr)
	}
	b.muPipe.Unlock()
	return nil
}

type pipeReader struct {
	io.Reader                // multi reader containing circbuf + pipe reader
	pr        *io.PipeReader // keep ref so it can be closed directly
	buf       *Buffer
}

func (r *pipeReader) Read(by []byte) (int, error) {
	n, err := r.Reader.Read(by)
	if err == io.ErrClosedPipe {
		return n, io.EOF
	}
	return n, err
}

func (r *pipeReader) Close() error {
	r.buf.Release(r)
	err := r.pr.Close()
	return err
}
