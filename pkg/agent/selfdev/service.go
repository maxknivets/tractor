package selfdev

import (
	"context"
	"os"
	"os/exec"
	"sync"

	"github.com/manifold/tractor/pkg/agent"
)

type Service struct {
	Agent *agent.Agent
}

func (s *Service) InitializeDaemon() error {
	return nil
}

func (s *Service) Serve(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("tsc", "-watch", "--preserveWatchOutput", "-p", "./extension")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}()
	wg.Wait()
}
