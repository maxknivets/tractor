package agentservice

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/manifold/qtalk/libmux/mux"
	"github.com/manifold/qtalk/qrpc"
	"github.com/manifold/tractor/pkg/agent"
)

// Service provides a QRPC server to connect, restart, and stop running
// workspaces.
type Service struct {
	Agent *agent.Agent
	api   qrpc.API
}

func (s *Service) InitializeDaemon() error {
	s.api = qrpc.NewAPI()
	s.api.HandleFunc("connect", s.Connect())
	s.api.HandleFunc("start", s.Start())
	s.api.HandleFunc("stop", s.Stop())
	return nil
}

func (s *Service) Serve(ctx context.Context) {
	server := &qrpc.Server{}
	l, err := mux.ListenUnix(s.Agent.SocketPath)
	if err != nil {
		fmt.Printf("error listening to unix://%s: %+v\n", s.Agent.SocketPath, err)
		return
	}

	s.periodicStatus()

	log.Printf("[server] unix://%s", s.Agent.SocketPath)
	if err = server.Serve(l, s.api); err != nil {
		fmt.Println(err)
	}
	os.Remove(s.Agent.SocketPath)
	return

}

func (s *Service) TerminateDaemon() error {
	s.Agent.Shutdown()
	return nil
}

func (s *Service) periodicStatus() {
	go func() {
		var lastMsg string
		for {
			time.Sleep(time.Second * 3)
			msg, err := wsStatus(s.Agent)
			if err != nil {
				log.Println("[workspaces]", err)
			}
			if lastMsg != msg && len(msg) > 0 {
				log.Println("[workspaces]", msg)
			}
			lastMsg = msg
		}
	}()
}

func wsStatus(a *agent.Agent) (string, error) {
	workspaces, err := a.Workspaces()
	if err != nil || len(workspaces) == 0 {
		return "", err
	}

	pairs := make([]string, len(workspaces))
	for i, ws := range workspaces {
		p, w := ws.BufferStatus()
		pairs[i] = fmt.Sprintf("%s=%s (%d pipe(s), %d written)",
			ws.Name, ws.Status, p, w)
	}
	return strings.Join(pairs, ", "), nil
}

func findWorkspace(a *agent.Agent, call *qrpc.Call) (*agent.Workspace, error) {
	var workspacePath string
	if err := call.Decode(&workspacePath); err != nil {
		return nil, err
	}
	log.Println("[qrpc]", call.Destination, workspacePath)

	if ws := a.Workspace(workspacePath); ws != nil {
		return ws, nil
	}

	return nil, fmt.Errorf("no workspace found for %q", workspacePath)
}