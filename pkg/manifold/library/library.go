package library

import (
	"runtime"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/progrium/prototypes/go-reflected"
)

var (
	registered []*RegisteredComponent
)

type RegisteredComponent struct {
	Type     reflected.Type
	Filepath string
	ID       string
}

func (rc *RegisteredComponent) New() manifold.Component {
	return NewComponent(rc.Type.Name(), rc.NewValue(), rc.ID)
}

func (rc *RegisteredComponent) NewValue() interface{} {
	return reflected.New(rc.Type).Interface()
}

func Register(v interface{}, id, filepath string) {
	if filepath == "" {
		_, filepath, _, _ = runtime.Caller(1)
	}
	registered = append(registered, &RegisteredComponent{
		Type:     reflected.ValueOf(v).Type(),
		Filepath: filepath,
		ID:       id,
	})
}

// deprecated
func Names() []string {
	var names []string
	for _, rc := range registered {
		if rc.ID != "" {
			continue
		}
		names = append(names, rc.Type.Name())
	}
	return names
}

func Registered() []*RegisteredComponent {
	r := make([]*RegisteredComponent, len(registered))
	copy(r, registered)
	return r
}

func Lookup(name string) *RegisteredComponent {
	for _, rc := range registered {
		if rc.Type.Name() == name {
			return rc
		}
	}
	return nil
}

func LookupID(id string) *RegisteredComponent {
	for _, rc := range registered {
		if rc.ID == id {
			return rc
		}
	}
	return nil
}
