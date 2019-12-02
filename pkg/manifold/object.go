package manifold

import "strings"

func NewObject(name string) Object {
	obj := &object{
		name:      name,
		observers: make(map[*ObjectObserver]struct{}),
	}
	obj.treeNode.object = obj
	return obj
}

type object struct {
	treeNode
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
