package manifold

import (
	"strings"
)

// observe treenode changes

func (o *object) SetParent(parent TreeNode) {
	o.treeNode.SetParent(parent)
	o.Notify(o, "Parent", o.parent, parent)
}

func (o *object) RemoveChildAt(idx int) TreeNode {
	removed := o.treeNode.RemoveChildAt(idx)
	o.Notify(o, "RemoveChildAt", removed, nil)
	return removed
}

func (o *object) InsertChildAt(idx int, child TreeNode) {
	o.treeNode.InsertChildAt(idx, child)
	o.Notify(o, "InsertChildAt", nil, child)
}

func (o *object) RemoveChild(child TreeNode) {
	o.treeNode.RemoveChild(child)
	o.Notify(o, "RemoveChild", child, nil)
}

func (o *object) AppendChild(child TreeNode) {
	o.treeNode.AppendChild(child)
	o.Notify(o, "AppendChild", nil, child)
}

// observe attributeset changes

func (o *object) SetAttribute(attr string, value interface{}) {
	prev := o.GetAttribute(attr)
	o.attributeset.SetAttribute(attr, value)
	o.Notify(o, attr, prev, value)
}

func (o *object) UnsetAttribute(attr string) {
	prev := o.GetAttribute(attr)
	o.attributeset.UnsetAttribute(attr)
	o.Notify(o, attr, prev, nil)
}

// observer implementation

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
