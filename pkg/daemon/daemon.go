package daemon

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Initializer is initialized before services are started. Returning
// an error will cancel the start of daemon services.
type Initializer interface {
	InitializeDaemon() error
}

// Terminator is terminated when the daemon gets a stop signal.
type Terminator interface {
	TerminateDaemon() error
}

type Service interface {
	Serve(ctx context.Context)
}

type Daemon struct {
	Initializers []Initializer
	Services     []Service
	Terminators  []Terminator
	Context      context.Context
}

func (d *Daemon) Run(ctx context.Context) error {
	// call initializers
	for _, i := range d.Initializers {
		if err := i.InitializeDaemon(); err != nil {
			return err
		}
	}

	// finish if no services
	if len(d.Services) == 0 {
		return nil
	}

	// setup terminators on stop signals
	termSigs := make(chan os.Signal, 1)
	signal.Notify(termSigs, os.Interrupt, os.Kill, syscall.SIGHUP)

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	d.Context = ctx

	termErr := make(chan error)
	go func() {
		select {
		case <-termSigs:
		case <-ctx.Done():
		}
		cancelFunc()
		for _, i := range d.Terminators {
			if err := i.TerminateDaemon(); err != nil {
				// TODO: better handling of multiple errors
				termErr <- err
				break
			}
		}
		close(termErr)
	}()

	var wg sync.WaitGroup
	for _, service := range d.Services {
		wg.Add(1)
		go func(s Service) {
			s.Serve(d.Context)
			wg.Done()
		}(service)
	}
	wg.Wait()

	return <-termErr
}
