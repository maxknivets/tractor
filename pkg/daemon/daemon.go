package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/manifold/tractor/pkg/registry"
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

// Service is run after the daemon is initialized.
type Service interface {
	Serve(ctx context.Context)
}

// Daemon is a top-level daemon lifecycle manager runs services given to it.
type Daemon struct {
	Initializers []Initializer
	Services     []Service
	Terminators  []Terminator
	Logger       log.Logger
	Context      context.Context
	state        int32
	cancel       context.CancelFunc
	termErrs     chan []error
}

// New builds a daemon configured to run a set of services. The services
// are populated with each other if they have fields that match anything
// that was passed in.
func New(services ...Service) *Daemon {
	d := &Daemon{}
	r, _ := registry.New(d)
	for _, s := range services {
		r.Register(s)
	}
	r.SelfPopulate()
	return d
}

// Run creates a daemon from services and runs it with a background context
func Run(services ...Service) error {
	d := New(services...)
	return d.Run(context.Background())
}

// Run executes the daemon lifecycle
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
		return errors.New("no services to run")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	d.Context = ctx
	d.cancel = cancelFunc
	d.termErrs = make(chan []error)

	// setup terminators on stop signals
	go TerminateOnSignal(d)
	go TerminateOnContextDone(d)

	var wg sync.WaitGroup
	for _, service := range d.Services {
		wg.Add(1)
		go func(s Service) {
			s.Serve(d.Context)
			wg.Done()
		}(service)
	}

	finished := make(chan bool)
	go func() {
		wg.Wait()
		close(finished)
	}()

	var errs []error
	select {
	case <-finished:
		errs = <-d.termErrs
	case errs = <-d.termErrs:
		fmt.Println("warning: unfinished services")
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Terminate cancels the daemon context and calls Terminators in reverse order
func (d *Daemon) Terminate() {
	if d == nil {
		return
	}

	if !atomic.CompareAndSwapInt32(&d.state, 1, 0) {
		return
	}

	if d.cancel != nil {
		d.cancel()
	}

	var errs []error
	for i := len(d.Terminators) - 1; i >= 0; i-- {
		if err := d.Terminators[i].TerminateDaemon(); err != nil {
			errs = append(errs, err)
		}
	}
	d.termErrs <- errs
}

// TerminateOnSignal waits for SIGINT, SIGHUP, SIGTERM, SIGKILL(?) to terminate the daemon.
func TerminateOnSignal(d *Daemon) {
	termSigs := make(chan os.Signal, 1)
	signal.Notify(termSigs, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGTERM)
	<-termSigs
	d.Terminate()
}

// TerminateOnContextDone waits for the deamon's context to be canceled.
func TerminateOnContextDone(d *Daemon) {
	<-d.Context.Done()
	d.Terminate()
}
