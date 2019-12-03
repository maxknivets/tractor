package manifold

import (
	"strings"
)

func newObject(name string) *object {
	obj := &object{
		name:         name,
		observers:    make(map[*ObjectObserver]struct{}),
		attributeset: attributeset(make(map[string]interface{})),
		componentlist: componentlist{
			components: make([]Component2, 0),
		},
	}
	obj.treeNode.object = obj
	obj.treeNode.children = make([]TreeNode, 0)
	return obj
}

func NewObject(name string) Object {
	return newObject(name)
}

type object struct {
	treeNode
	componentlist
	attributeset
	sys       System
	name      string
	observers map[*ObjectObserver]struct{}
}

func (o *object) SetParent(parent TreeNode) {
	o.treeNode.SetParent(parent)
	o.Notify(o, "Parent", o.parent, parent)
}

func (o *object) Name() string {
	return o.name
}

func (o *object) Observe(obs *ObjectObserver) {
	o.observers[obs] = struct{}{}
}

func (o *object) Unobserve(obs *ObjectObserver) {
	delete(o.observers, obs)
}

func (o *object) Notify(obj Object, path string, old, new interface{}) {
	for obs := range o.observers {
		if strings.HasPrefix(path, obs.Path) {
			obs.OnChange(obj, path, old, new)
		}
	}
	if o.parent != nil {
		o.parent.Object().Notify(obj, path, old, new)
	}
}

func (o *object) System() System {
	if o.sys != nil {
		return o.sys
	}
	return o.Root().Object().System()
}
