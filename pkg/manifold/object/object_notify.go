package object

import (
	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/misc/notify"
)

// observe component list changes

func (o *object) AppendComponent(com manifold.Component) {
	o.componentlist.AppendComponent(com)
	com.SetContainer(o)
	o.UpdateRegistry()
	o.registry.Populate(com.Pointer())
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Components",
		New:    com,
	})

}

func (o *object) RemoveComponent(com manifold.Component) {
	o.componentlist.RemoveComponent(com)
	o.UpdateRegistry()
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Components",
		Old:    com,
	})
	if o.main == com {
		o.main = nil
	}
}

func (o *object) InsertComponentAt(idx int, com manifold.Component) {
	o.componentlist.InsertComponentAt(idx, com)
	com.SetContainer(o)
	o.UpdateRegistry()
	o.registry.Populate(com.Pointer())
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Components",
		New:    com,
	})
}

func (o *object) RemoveComponentAt(idx int) manifold.Component {
	c := o.componentlist.RemoveComponentAt(idx)
	o.UpdateRegistry()
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Components",
		Old:    c,
	})
	if o.main == c {
		o.main = nil
	}
	return c
}

// observe attributeset changes

func (o *object) SetAttribute(attr string, value interface{}) {
	prev := o.GetAttribute(attr)
	if prev != value {
		o.attributeset.SetAttribute(attr, value)
		notify.Send(o, manifold.ObjectChange{
			Object: o,
			Path:   "--" + attr,
			Old:    prev,
			New:    value,
		})
	}
}

func (o *object) UnsetAttribute(attr string) {
	prev := o.GetAttribute(attr)
	if prev != nil {
		o.attributeset.UnsetAttribute(attr)
		notify.Send(o, manifold.ObjectChange{
			Object: o,
			Path:   "--" + attr,
			Old:    prev,
		})
	}
}
