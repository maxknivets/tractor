package daemon

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
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
	state        int32
	cancel       context.CancelFunc
	errs         chan []error
}

func New(services ...Service) *Daemon {
	d := &Daemon{Services: services}
	for _, s := range services {
		if i, ok := s.(Initializer); ok {
			d.Initializers = append(d.Initializers, i)
		}
		if t, ok := s.(Terminator); ok {
			d.Terminators = append(d.Terminators, t)
		}
	}
	return d
}

func Run(services ...Service) error {
	d := New(services...)
	return d.Run(context.Background())
}

func (d *Daemon) Run(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&d.state, 0, 1) {
		return errors.New("already running")
	}

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

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	d.Context = ctx
	d.cancel = cancelFunc
	d.errs = make(chan []error)

	// setup terminators on stop signals
	CancelBySignal(d)
	CancelByContext(d)

	var wg sync.WaitGroup
	for _, service := range d.Services {
		wg.Add(1)
		go func(s Service) {
			s.Serve(d.Context)
			wg.Done()
		}(service)
	}
	wg.Wait()
	errs := <-d.errs
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (d *Daemon) Terminate() {
	if !atomic.CompareAndSwapInt32(&d.state, 1, 0) {
		return
	}

	if d.cancel != nil {
		d.cancel()
	}
	var errs []error
	for _, i := range d.Terminators {
		if err := i.TerminateDaemon(); err != nil {
			errs = append(errs, err)
		}
	}
	d.errs <- errs
}

func CancelBySignal(d *Daemon) {
	go func() {
		termSigs := make(chan os.Signal, 1)
		signal.Notify(termSigs, os.Interrupt, os.Kill, syscall.SIGHUP)
		<-termSigs
		d.Terminate()
	}()
}

func CancelByContext(d *Daemon) {
	go func() {
		<-d.Context.Done()
		d.Terminate()
	}()
}
