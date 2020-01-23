package rpc

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/manifold/qtalk/qrpc"
	"github.com/manifold/tractor/pkg/manifold/library"
	"github.com/manifold/tractor/pkg/manifold/object"
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
		com := n.Component(params.Component)
		n.RemoveComponent(com)
		if com.ID() == n.ID() {
			if err := s.State.Image.DestroyObjectPackage(n); err != nil {
				fmt.Println(err)
			}
		}
		s.updateView()
		r.Return(nil)
	}
}

func (s *Service) ReloadComponent() func(qrpc.Responder, *qrpc.Call) {
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
		com := n.Component(params.Component)
		if e, ok := com.Pointer().(disabler); ok {
			e.OnDisable()
		}
		if e, ok := com.Pointer().(enabler); ok {
			e.OnEnable()
		}
		s.updateView()
		r.Return(nil)
	}
}

type enabler interface {
	OnEnable()
}
type disabler interface {
	OnDisable()
}

func (s *Service) AddDelegate() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params NodeParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		obj := s.State.Root.FindID(params.ID)
		if obj == nil {
			r.Return(nil)
			return
		}
		r.Return(s.State.Image.CreateObjectPackage(obj))
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
			n.SetName(*params.Name)
		}
		// if params.Active != nil {
		// 	n.Active = *params.Active
		// }
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
		n := s.State.Root.FindChild(path)
		localPath := path[len(n.Path())+1:]
		// TODO: support args+ret
		n.CallMethod(localPath, nil, nil)
		s.updateView()
		r.Return(nil)
	}
}

// func (s *Service) SetExpression() func(qrpc.Responder, *qrpc.Call) {
// 	return func(r qrpc.Responder, c *qrpc.Call) {
// 		var params SetValueParams
// 		err := c.Decode(&params)
// 		if err != nil {
// 			r.Return(err)
// 			return
// 		}
// 		n := s.State.Root.FindChild(params.Path)
// 		localPath := params.Path[len(n.Path())+1:]
// 		// n.SetExpression(localPath, params.Value.(string))
// 		s.updateView()
// 		r.Return(nil)
// 	}
// }

func (s *Service) SetValue() func(qrpc.Responder, *qrpc.Call) {
	return func(r qrpc.Responder, c *qrpc.Call) {
		var params SetValueParams
		err := c.Decode(&params)
		if err != nil {
			r.Return(err)
			return
		}
		n := s.State.Root.FindChild(params.Path)
		//fmt.Println(n, params)
		localPath := params.Path[len(n.Path())+1:]
		switch {
		case params.IntValue != nil:
			n.SetField(localPath, *params.IntValue)
		case params.RefValue != nil:
			refPath := filepath.Dir(*params.RefValue) // TODO: support subfields
			refNode := s.State.Root.FindChild(refPath)
			parts := strings.SplitN(localPath, "/", 2)
			refType := n.Component(parts[0]).FieldType(parts[1])
			if refNode != nil {
				typeSelector := (*params.RefValue)[len(refNode.Path())+1:]
				c := refNode.Component(typeSelector)
				if c != nil {
					n.SetField(localPath, c)
				} else {
					// interface reference
					ptr := reflect.New(refType)
					refNode.ValueTo(ptr)
					if ptr.IsValid() {
						n.SetField(localPath, reflect.Indirect(ptr).Interface())
					}
				}
			}
		default:
			n.SetField(localPath, params.Value)
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
		v := library.Lookup(params.Name).New()
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
		s.State.Root.RemoveID(id)
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
		n := object.New(params.Name)
		p.AppendChild(n)
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
