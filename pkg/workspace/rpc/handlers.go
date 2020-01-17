package rpc

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/manifold/qtalk/qrpc"
	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/workspace/state"
)

type AppendNodeParams struct {
	ID   string
	Name string
}

type SetValueParams struct {
	Path     string
	Value    interface{}
	IntValue *int
	RefValue *string
}

type RemoveComponentParams struct {
	ID        string
	Component string
}

type NodeParams struct {
	ID     string
	Name   *string
	Active *bool
}

type DelegateParams struct {
	ID       string
	Contents string
}

type MoveNodeParams struct {
	ID    string
	Index int
}

func (s *Service) Reload() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		s.updateView()
		r.Return(nil)
	}
}

// api.HandleFunc("repl", func(r qrpc.Responder, c *qrpc.Call) {
// 	var params DelegateParams
// 	_ = c.Decode(&params)
// 	// ^^ TODO: make sure this isn't necessary before hijacking
// 	ch, err := r.Hijack(nil)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	repl := repl.NewREPL(func(v interface{}) {
// 		fmt.Fprintf(ch, "%s\n", v)
// 	})
// 	repl.Run(ch, ch, map[string]interface{}{
// 		"Root": root,
// 	})
// })

func (s *Service) RemoveComponent() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params RemoveComponentParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindID(params.ID)
		if n == nil {
			r.Return(fmt.Errorf("unable to find node: %s", params.ID))
			return
		}
		n.RemoveComponent(params.Component)
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) AddDelegate() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params NodeParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindID(params.ID)
		if n == nil {
			return
		}
		r.Return(state.CreateDelegate(n))
	}
}

func (s *Service) SelectNode() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var id string
		err := c.Decode(&id)
		if err != nil {
			r.Return(err)
			return
		}
		s.viewState.SelectedNode = id
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) UpdateNode() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params NodeParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindID(params.ID)
		if n == nil {
			return
		}
		if params.Name != nil {
			n.Name = *params.Name
		}
		if params.Active != nil {
			n.Active = *params.Active
		}
		n.Sync()
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) CallMethod() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var path string
		err := c.Decode(&path)
		if err != nil {
			r.Return(err)
			return
		}
		if path == "" {
			return
		}
		n := s.State.Root.FindNode(path)
		localPath := path[len(n.FullPath())+1:]
		n.CallMethod(localPath)
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) SetExpression() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params SetValueParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindNode(params.Path)
		localPath := params.Path[len(n.FullPath())+1:]
		n.SetExpression(localPath, params.Value.(string))
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) SetValue() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params SetValueParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindNode(params.Path)
		localPath := params.Path[len(n.FullPath())+1:]
		switch {
		case params.IntValue != nil:
			n.SetValue(localPath, *params.IntValue)
		case params.RefValue != nil:
			refPath := filepath.Dir(*params.RefValue) // TODO: support subfields
			refNode := s.State.Root.FindNode(refPath)
			refType := n.Field(localPath)
			if refNode != nil {
				typeSelector := (*params.RefValue)[len(refNode.FullPath())+1:]
				c := refNode.Component(typeSelector)
				if c != nil {
					n.SetValue(localPath, c)
				} else {
					// interface reference
					ptr := reflect.New(refType)
					refNode.Registry.ValueTo(ptr)
					if ptr.IsValid() {
						n.SetValue(localPath, reflect.Indirect(ptr).Interface())
					}
				}
			}
		default:
			n.SetValue(localPath, params.Value)
		}
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) AppendComponent() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params AppendNodeParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		if params.Name == "" {
			return
		}
		p := s.State.Root.FindID(params.ID)
		if p == nil {
			p = s.State.Root
		}
		v := manifold.NewComponent(params.Name)
		p.AppendComponent(v)
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) DeleteNode() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var id string
		err := c.Decode(&id)
		if err != nil {
			r.Return(err)
			return
		}
		if id == "" {
			return
		}
		n := s.State.Root.FindID(id)
		n.Remove()
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) AppendNode() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params AppendNodeParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		if params.Name == "" {
			return
		}
		p := s.State.Root.FindID(params.ID)
		if p == nil {
			p = s.State.Root
		}
		n := manifold.NewNode(params.Name)
		p.Append(n)
		n.Sync()
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) MoveNode() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params MoveNodeParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindID(params.ID)
		if n == nil {
			return
		}
		n.SetSiblingIndex(params.Index)
		// n.Sync()
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) Subscribe() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		s.clients[c.Caller] = "state"
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) SelectProject() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var name string
		err := c.Decode(&name)
		if err != nil {
			r.Return(err)
			return
		}
		s.viewState.CurrentProject = name
		s.updateView()
		r.Return(nil)
	}
}
