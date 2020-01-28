package object

import (
	"fmt"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/misc/notify"
)

type privTreeNode interface {
	manifold.Object
	setChildren(ch []manifold.Object)
}

func (o *object) Root() manifold.Object {
	if o.parent != nil {
		return o.parent.Root()
	}
	return o
}

func (o *object) Parent() manifold.Object {
	return o.parent
}

func (o *object) SetParent(obj manifold.Object) {
	old := o.parent
	if old != obj {
		o.parent = obj
		notify.Send(o, manifold.ObjectChange{
			Object: o,
			Path:   "::Parent",
			Old:    old,
			New:    obj,
		})
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
		return fmt.Errorf("parent type %T must implement setChildren([]manifold.Object)", o.parent)
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
	newChildren := make([]manifold.Object, idx+1)
	copy(newChildren, oldChildren[:idx])
	newChildren[idx] = o
	parent.setChildren(append(newChildren, oldChildren[idx:]...))

	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::SiblingIndex",
		Old:    oldIndex,
		New:    idx,
	})
	return nil
}

func (o *object) setChildren(ch []manifold.Object) {
	o.children = ch
}

func (o *object) NextSibling() manifold.Object {
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

func (o *object) PreviousSibling() manifold.Object {
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

func (o *object) Children() []manifold.Object {
	ch := make([]manifold.Object, len(o.children))
	copy(ch, o.children)
	return ch
}

func (o *object) RemoveChildAt(idx int) manifold.Object {
	child := o.ChildAt(idx)
	if child == nil {
		return nil
	}

	if len(o.children) > idx {
		o.children = append(o.children[:idx], o.children[idx+1:]...)
	} else {
		o.children = o.children[:idx]
	}
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Children",
		Old:    child,
	})
	return child
}

func (o *object) InsertChildAt(idx int, child manifold.Object) {
	if idx < 0 {
		panic(fmt.Sprintf("cannot insert child to index: %d", idx))
	}

	defer notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Children",
		New:    child,
	})

	child.SetParent(o)
	if idx >= len(o.children) {
		o.AppendChild(child)
		return
	}

	o.children = append(o.children[:idx],
		append([]manifold.Object{child}, o.children[idx:]...)...)
}

func (o *object) RemoveChild(child manifold.Object) {
	idx := o.childIndex(child)
	if idx < 0 {
		return
	}
	o.RemoveChildAt(idx)
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Children",
		Old:    child,
	})
}

func (o *object) AppendChild(child manifold.Object) {
	child.SetParent(o)
	o.children = append(o.children, child)
	notify.Send(o, manifold.ObjectChange{
		Object: o,
		Path:   "::Children",
		New:    child,
	})
}

func (o *object) ChildAt(idx int) manifold.Object {
	if idx > -1 && len(o.children) > idx {
		return o.children[idx]
	}
	return nil
}

func (o *object) childIndex(child manifold.Object) int {
	for i, c := range o.children {
		if c == child {
			return i
		}
	}
	return -1
}
