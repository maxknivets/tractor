package agent

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/manifold/qtalk/libmux/mux"
	"github.com/manifold/qtalk/qrpc"
)

func ListenAndServe(a *Agent) error {
	api := qrpc.NewAPI()
	api.HandleFunc("connect", func(r qrpc.Responder, c *qrpc.Call) {
		ws, err := findWorkspace(a, c)
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
	})

	api.HandleFunc("start", func(r qrpc.Responder, c *qrpc.Call) {
		ws, err := findWorkspace(a, c)
		if err != nil {
			r.Return(err)
			return
		}

		// TODO: shouldn't stream logs or block, but maybe we show a snippet? -JL

		_, err = ws.Start()
		if err != nil {
			r.Return(err)
			return
		}
	})

	api.HandleFunc("stop", func(r qrpc.Responder, c *qrpc.Call) {
		ws, err := findWorkspace(a, c)
		if err != nil {
			r.Return(err)
			return
		}
		ws.Stop()

		r.Return(fmt.Sprintf("workspace %q stopped", ws.Name))
	})

	go func() {
		var lastMsg string
		for {
			time.Sleep(time.Second * 3)
			msg, err := wsStatus(a)
			if err != nil {
				log.Println("[workspaces]", err)
			}
			if lastMsg != msg && len(msg) > 0 {
				log.Println("[workspaces]", msg)
			}
			lastMsg = msg
		}
	}()

	server := &qrpc.Server{}
	l, err := mux.ListenUnix(a.SocketPath)
	if err != nil {
		return err
	}

	log.Printf("[server] unix://%s", a.SocketPath)
	err = server.Serve(l, api)
	os.Remove(a.SocketPath)
	return err
}

func wsStatus(a *Agent) (string, error) {
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

func findWorkspace(a *Agent, call *qrpc.Call) (*Workspace, error) {
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
