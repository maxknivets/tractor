package rpc

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/manifold/qtalk/libmux/mux"
	"github.com/manifold/qtalk/qrpc"
	"github.com/manifold/tractor/pkg/misc/logging"
	"github.com/manifold/tractor/pkg/workspace/state"
	"github.com/manifold/tractor/pkg/workspace/view"
)

type Service struct {
	Protocol   string
	ListenAddr string

	Log   logging.Logger
	State *state.Service

	viewState *view.State
	clients   map[qrpc.Caller]string
	api       qrpc.API
	l         mux.Listener
}

func (s *Service) updateView() {
	// TODO: mutex, etc
	s.viewState.Update(s.State.Root)
	for client, callback := range s.clients {
		_, err := client.Call(callback, s.viewState, nil)
		if err != nil {
			delete(s.clients, client)
			log.Println(err)
		}
	}
}

func (s *Service) InitializeDaemon() (err error) {
	if s.l, err = muxListenTo(s.Protocol, s.ListenAddr); err != nil {
		return err
	}

	s.clients = make(map[qrpc.Caller]string)
	s.viewState = view.New(s.State.Root)

	s.api = qrpc.NewAPI()
	s.api.HandleFunc("reload", s.Reload())
	s.api.HandleFunc("selectNode", s.SelectNode())
	s.api.HandleFunc("removeComponent", s.RemoveComponent())
	s.api.HandleFunc("selectProject", s.SelectProject())
	s.api.HandleFunc("moveNode", s.MoveNode())
	s.api.HandleFunc("subscribe", s.Subscribe())
	s.api.HandleFunc("appendNode", s.AppendNode())
	s.api.HandleFunc("deleteNode", s.DeleteNode())
	s.api.HandleFunc("appendComponent", s.AppendComponent())
	s.api.HandleFunc("setValue", s.SetValue())
	// s.api.HandleFunc("setExpression", s.SetExpression())
	s.api.HandleFunc("callMethod", s.CallMethod())
	s.api.HandleFunc("updateNode", s.UpdateNode())
	s.api.HandleFunc("addDelegate", s.AddDelegate())

	return nil
}

func (s *Service) Serve(ctx context.Context) {
	server := &qrpc.Server{}
	s.Log.Infof("[workspace] %s://%s", s.Protocol, s.ListenAddr)
	if err := server.Serve(s.l, s.api); err != nil {
		fmt.Println(err)
	}
	if s.Protocol == "unix" {
		os.Remove(s.ListenAddr)
	}
}

func (s *Service) TerminateDaemon() error {
	for client, _ := range s.clients {
		client.Call("shutdown", nil, nil)
	}
	if s.Protocol == "unix" {
		os.Remove(s.ListenAddr)
	}
	return nil
}

func muxListenTo(proto, addr string) (mux.Listener, error) {
	switch proto {
	case "websocket":
		return mux.ListenWebsocket(addr)
	case "unix":
		return mux.ListenUnix(addr)
	}

	return nil, fmt.Errorf("cannot connect to %s, unknown protocol %q", addr, proto)
}
