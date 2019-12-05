package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/console"
	"github.com/manifold/tractor/pkg/daemon"
	"github.com/manifold/tractor/pkg/log"
	"github.com/manifold/tractor/pkg/log/std"
	"github.com/manifold/tractor/pkg/registry"
)

type tempService struct {
	Log log.InfoLogger
}

func (s *tempService) InitializeDaemon() error {
	s.Log.Info("Hello!")
	return nil
}

func (s *tempService) TerminateDaemon() error {
	s.Log.Info("Goodbye!")
	return nil
}

func (s *tempService) Serve(ctx context.Context) {
	for {
		<-time.After(2 * time.Second)
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.Log.Infof("serving: %d", time.Now().Unix())
	}
}

func main() {
	debugService := new(tempService)
	buf, err := agent.NewBuffer(1024)
	if err != nil {
		panic(err)
	}
	logger := std.NewLogger("", buf)

	pipe := buf.Pipe()
	var wg sync.WaitGroup
	c := &console.Console{
		Output: os.Stdout,
	}
	c.SystemOutput("Hello")
	wg.Add(1)
	go c.LineReader(&wg, "BLAH", 0, pipe, false)

	r := registry.New()
	r.Register(
		registry.Ref(debugService),
		registry.Ref(logger),
	)
	r.Populate(debugService)

	d := new(daemon.Daemon)
	r.Populate(d)

	d.Run(context.Background())
}
