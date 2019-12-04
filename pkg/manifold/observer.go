package manifold

import (
	"strings"
)

// observe treenode changes

func (o *object) SetParent(parent TreeNode) {
	o.treeNode.SetParent(parent)
	notify(o, o, "Parent", o.parent, parent)
}

func (o *object) SetSiblingIndex(idx int) error {
	old := o.treeNode.SiblingIndex()
	if err := o.treeNode.SetSiblingIndex(idx); err != nil {
		return err
	}

	notify(o, o, "SetSiblingIndex", old, idx)
	return nil
}

func (o *object) RemoveChildAt(idx int) TreeNode {
	removed := o.treeNode.RemoveChildAt(idx)
	notify(o, o, "RemoveChildAt", removed, nil)
	return removed
}

func (o *object) InsertChildAt(idx int, child TreeNode) {
	o.treeNode.InsertChildAt(idx, child)
	notify(o, o, "InsertChildAt", nil, child)
}

func (o *object) RemoveChild(child TreeNode) {
	o.treeNode.RemoveChild(child)
	notify(o, o, "RemoveChild", child, nil)
}

func (o *object) AppendChild(child TreeNode) {
	o.treeNode.AppendChild(child)
	notify(o, o, "AppendChild", nil, child)
}

// observe attributeset changes

func (o *object) SetAttribute(attr string, value interface{}) {
	prev := o.GetAttribute(attr)
	o.attributeset.SetAttribute(attr, value)
	notify(o, o, attr, prev, value)
}

func (o *object) UnsetAttribute(attr string) {
	prev := o.GetAttribute(attr)
	o.attributeset.UnsetAttribute(attr)
	notify(o, o, attr, prev, nil)
}

// observer implementation

func (o *object) Observe(obs *ObjectObserver) {
	o.observers[obs] = struct{}{}
}

func (o *object) Unobserve(obs *ObjectObserver) {
	delete(o.observers, obs)
}

func notify(sender *object, changed Object, path string, old, new interface{}) {
	for obs := range sender.observers {
		if strings.HasPrefix(path, obs.Path) {
			obs.OnChange(changed, path, old, new)
		}
	}

	if sender.parent == nil {
		return
	}

	parent, ok := sender.parent.Object().(*object)
	if ok && parent == nil {
		return
	}

	notify(parent, changed, path, old, new)
}
