package console

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/manifold/tractor/pkg/misc/logging"
	"github.com/manifold/tractor/pkg/misc/logging/std"
)

// Output lets you redirect the output the console is set up with.
var Output = os.Stdout

// Service provides a logging service and console output
type Service struct {
	logging.Logger

	logreader io.Reader
	logwriter io.WriteCloser
	console   *LineWriter
	mu        sync.Mutex
	idx       int
}

// New ...
func New() *Service {
	s := &Service{}
	s.logreader, s.logwriter = io.Pipe()
	s.Logger = std.NewLogger("", s.logwriter)
	s.console = &LineWriter{
		Output: Output,
	}
	go s.console.LineReader("agent", -1, s.logreader, false)
	return s
}

// Serve ...
func (s *Service) Serve(ctx context.Context) {
	s.console.Wait()
}

// TerminateDaemon ...
func (s *Service) TerminateDaemon() error {
	return s.logwriter.Close()
}

// NewReader ...
func (s *Service) NewReader(name string, reader io.Reader, isError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.idx
	s.idx++

	go s.console.LineReader(name, current, reader, isError)
}

func (s *Service) NewPipe(name string) io.WriteCloser {
	pr, pw := io.Pipe()
	s.NewReader(name, pr, false)
	return pw
}
