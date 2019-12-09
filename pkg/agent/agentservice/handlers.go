package agentservice

import (
	"fmt"
	"io"

	"github.com/manifold/qtalk/qrpc"
)

func (s *Service) Connect() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		ws, err := findWorkspace(s.Agent, c)
		if err != nil {
			r.Return(err)
			return
		}

		out, err := ws.Connect()
		if err != nil {
			r.Return(err)
			return
		}

		ch, err := r.Hijack(ws.SocketPath)
		if err != nil {
			out.Close()
			r.Return(err)
			return
		}

		_, err = io.Copy(ch, out)
		ch.Close()
		out.Close()

		if err == io.ErrClosedPipe {
			r.Return(err)
			return
		}
	}
}

func (s *Service) Start() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		ws, err := findWorkspace(s.Agent, c)
		if err != nil {
			r.Return(err)
			return
		}

		// TODO: shouldn't stream logs or block, but maybe we show a snippet? -JL
		if err := ws.Start(); err != nil {
			r.Return(err)
			return
		}
		r.Return(fmt.Sprintf("workspace %q started", ws.Name))
	}
}

func (s *Service) Stop() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		ws, err := findWorkspace(s.Agent, c)
		if err != nil {
			r.Return(err)
			return
		}
		ws.Stop()

		r.Return(fmt.Sprintf("workspace %q stopped", ws.Name))
	}
}
