package manifold

import (
	"fmt"
)

type privTreeNode interface {
	Object
	setChildren(ch []Object)
}

type treeNode struct {
	object Object
}

func (o *object) Root() Object {
	if o.parent != nil {
		return o.parent.Root()
	}
	return o
}

func (o *object) Parent() Object {
	return o.parent
}

func (o *object) SetParent(obj Object) {
	old := o.parent
	if old != obj {
		o.parent = obj
		o.notify(o, "::Parent", old, obj)
	}
}

func (o *object) SiblingIndex() int {
	if o.parent == nil {
		return 0
	}
	for i, c := range o.parent.Children() {
		if c == o {
			return i
		}
	}
	return 0
}

func (o *object) SetSiblingIndex(idx int) error {
	if o.parent == nil {
		return nil
	}
	if idx < 0 {
		return fmt.Errorf("index must be >= 0, got: %d", idx)
	}

	parent, ok := o.parent.(privTreeNode)
	if !ok {
		return fmt.Errorf("parent type %T must implement setChildren([]Object)", o.parent)
	}

	siblings := parent.Children()
	if ls := len(siblings); idx >= ls {
		return fmt.Errorf("index must be < %d sibling(s), got: %d", ls, idx)
	}

	oldIndex := o.SiblingIndex()
	if oldIndex == idx {
		return nil
	}

	oldChildren := append(siblings[:oldIndex], siblings[oldIndex+1:]...)
	newChildren := make([]Object, idx+1)
	copy(newChildren, oldChildren[:idx])
	newChildren[idx] = o
	parent.setChildren(append(newChildren, oldChildren[idx:]...))

	o.notify(o, "::SiblingIndex", oldIndex, idx)
	return nil
}

func (o *object) setChildren(ch []Object) {
	o.children = ch
}

func (o *object) NextSibling() Object {
	if o.parent == nil {
		return nil
	}

	next := o.SiblingIndex() + 1
	siblings := o.parent.Children()
	if next < len(siblings) {
		return siblings[next]
	}
	return nil
}

func (o *object) PreviousSibling() Object {
	if o.parent == nil {
		return nil
	}

	siblings := o.parent.Children()
	if len(siblings) == 0 {
		return nil
	}

	prev := o.SiblingIndex() - 1
	if prev < 0 {
		return nil
	}
	return siblings[prev]
}

func (o *object) Children() []Object {
	ch := make([]Object, len(o.children))
	copy(ch, o.children)
	return ch
}

func (o *object) RemoveChildAt(idx int) Object {
	child := o.ChildAt(idx)
	if child == nil {
		return nil
	}

	if len(o.children) > idx {
		o.children = append(o.children[:idx], o.children[idx+1:]...)
	} else {
		o.children = o.children[:idx]
	}
	o.notify(o, "::Children", child, nil)
	return child
}

func (o *object) InsertChildAt(idx int, child Object) {
	if idx < 0 {
		panic(fmt.Sprintf("cannot insert child to index: %d", idx))
	}

	defer o.notify(o, "::Children", nil, child)

	child.SetParent(o)
	if idx >= len(o.children) {
		o.AppendChild(child)
		return
	}

	o.children = append(o.children[:idx],
		append([]Object{child}, o.children[idx:]...)...)
}

func (o *object) RemoveChild(child Object) {
	idx := o.childIndex(child)
	if idx < 0 {
		return
	}
	o.RemoveChildAt(idx)
	o.notify(o, "::Children", child, nil)
}

func (o *object) AppendChild(child Object) {
	child.SetParent(o)
	o.children = append(o.children, child)
	o.notify(o, "::Children", nil, child)
}

func (o *object) ChildAt(idx int) Object {
	if idx > -1 && len(o.children) > idx {
		return o.children[idx]
	}
	return nil
}

func (o *object) childIndex(child Object) int {
	for i, c := range o.children {
		if c == child {
			return i
		}
	}
	return -1
}
