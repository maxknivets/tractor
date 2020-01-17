package state

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/misc/logging"
)

type Service struct {
	Protocol   string
	ListenAddr string

	Log  logging.Logger
	Root *manifold.Node
}

func (s *Service) InitializeDaemon() (err error) {
	s.Root, err = LoadHierarchy()
	if err != nil {
		return err
	}

	// TODO: deprecated, remove
	manifold.Walk(s.Root, func(n *manifold.Node) {
		for _, com := range n.Components {
			if initializer, ok := com.Ref.(preInitializer); ok {
				initializer.PreInitialize()
			}
		}
	})

	manifold.Walk(s.Root, func(n *manifold.Node) {
		for _, com := range n.Components {
			if initializer, ok := com.Ref.(initializer); ok {
				if err := initializer.Initialize(); err != nil {
					log.Print(err)
				}
			}
		}
	})

	s.Root.Observe(&manifold.NodeObserver{
		OnChange: func(node *manifold.Node, path string, old, new interface{}) {
			if path == "Name" && node.Dir != "" {
				newDir := filepath.Join(filepath.Dir(node.Dir), new.(string))
				if node.Dir != newDir {
					// TODO: do not break abstraction, have workspace handle this
					if err := os.Rename(node.Dir, newDir); err != nil {
						log.Fatal(err)
					}
				}
			}
			s.Snapshot()
		},
	})

	return nil
}

func (s *Service) TerminateDaemon() error {
	return s.Snapshot()
}

func (s *Service) Serve(ctx context.Context) {
	<-ctx.Done()
}

func (s *Service) Snapshot() error {
	// TODO: Log errors?
	return SaveHierarchy(s.Root)
}

type preInitializer interface {
	PreInitialize()
}

type initializer interface {
	Initialize() error
}
