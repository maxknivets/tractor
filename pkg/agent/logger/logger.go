package logger

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/manifold/tractor/pkg/console"
	"github.com/manifold/tractor/pkg/logging"
	"github.com/manifold/tractor/pkg/logging/std"
)

// Output lets you redirect the output the console is set up with.
var Output = os.Stdout

// Service provides a logging service and console output
type Service struct {
	logging.Logger

	logreader io.Reader
	logwriter io.WriteCloser
	console   *console.Console
	wg        sync.WaitGroup
	idx       int
}

// New ...
func New() *Service {
	s := &Service{}
	s.logreader, s.logwriter = io.Pipe()
	s.Logger = std.NewLogger("", s.logwriter)
	s.console = &console.Console{
		Output: Output,
	}
	s.wg.Add(1)
	go s.console.LineReader(&s.wg, "agent", -1, s.logreader, false)
	return s
}

// Serve ...
func (s *Service) Serve(ctx context.Context) {
	// TODO: cancel line readers with context
	s.wg.Wait()
}

// TerminateDaemon ...
func (s *Service) TerminateDaemon() error {
	return s.logwriter.Close()
}

// NewReader ...
func (s *Service) NewReader(name string, reader io.Reader, isError bool) {
	s.wg.Add(1)
	// TODO: close these on terminate
	go s.console.LineReader(&s.wg, name, s.idx, reader, isError)
	s.idx++
}

func (s *Service) NewPipe(name string) io.WriteCloser {
	pr, pw := io.Pipe()
	s.NewReader(name, pr, false)
	return pw
}
