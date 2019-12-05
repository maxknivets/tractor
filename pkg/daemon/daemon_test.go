package daemon

import (
	"context"
	"testing"

	"github.com/manifold/tractor/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type namedService struct {
	Name string

	mock.Mock
}

func (s *namedService) InitializeDaemon() error {
	args := s.Called()
	return args.Error(0)
}

func (s *namedService) TerminateDaemon() error {
	args := s.Called()
	return args.Error(0)
}

func (s *namedService) Serve(ctx context.Context) {
	s.Called(ctx)
	return
}

type initService struct {
	mock.Mock
}

func (s *initService) InitializeDaemon() error {
	args := s.Called()
	return args.Error(0)
}

type simpleService struct {
	mock.Mock
}

func (s *simpleService) Serve(ctx context.Context) {
	s.Called(ctx)
	return
}

func TestDaemon(t *testing.T) {
	s1 := new(initService)
	s2 := new(simpleService)
	s3 := &namedService{Name: "namedservice1"}
	s4 := &namedService{Name: "namedservice2"}

	s1.On("InitializeDaemon").Return(nil)
	s3.On("InitializeDaemon").Return(nil)
	s4.On("InitializeDaemon").Return(nil)

	s2.On("Serve", mock.Anything).Return()
	s3.On("Serve", mock.Anything).Return()
	s4.On("Serve", mock.Anything).Return()

	s3.On("TerminateDaemon").Return(nil)
	s4.On("TerminateDaemon").Return(nil)

	r := registry.New()
	require.Nil(t, r.Register(
		registry.Ref(s1),
		registry.Ref(s2),
		registry.Ref(s3),
		registry.Ref(s4),
	))

	d := &Daemon{}
	r.Populate(d)

	assert.Len(t, d.Initializers, 3)
	assert.Len(t, d.Terminators, 2)
	assert.Len(t, d.Services, 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d.Run(ctx)

	s1.AssertExpectations(t)
	s2.AssertExpectations(t)
	s3.AssertExpectations(t)
	s4.AssertExpectations(t)
}
