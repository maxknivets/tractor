package manifold

import (
	"fmt"
)

type privTreeNode interface {
	TreeNode
	setChildren(ch []TreeNode)
}

type treeNode struct {
	object   Object
	parent   TreeNode
	children []TreeNode
}

func (t *treeNode) Object() Object {
	return t.object
}

func (t *treeNode) Root() TreeNode {
	if t.parent != nil {
		return t.parent.Root()
	}
	return t
}

func (t *treeNode) Parent() TreeNode {
	return t.parent
}

func (t *treeNode) SetParent(obj TreeNode) {
	t.parent = obj
}

func (t *treeNode) SiblingIndex() int {
	if t.parent == nil {
		return 0
	}
	for i, c := range t.parent.ChildNodes() {
		if c.Object() == t.Object() {
			return i
		}
	}
	return 0
}

func (t *treeNode) SetSiblingIndex(idx int) error {
	if t.parent == nil {
		return nil
	}
	if idx < 0 {
		return fmt.Errorf("index must be >= 0, got: %d", idx)
	}

	parent, ok := t.parent.(privTreeNode)
	if !ok {
		return fmt.Errorf("parent type %T must implement setChildren([]TreeNode)", t.parent)
	}

	siblings := parent.ChildNodes()
	if ls := len(siblings); idx >= ls {
		return fmt.Errorf("index must be < %d sibling(s), got: %d", ls, idx)
	}

	oldIndex := t.SiblingIndex()
	oldChildren := append(siblings[:oldIndex], siblings[oldIndex+1:]...)
	newChildren := make([]TreeNode, idx+1)
	copy(newChildren, oldChildren[:idx])
	newChildren[idx] = t
	parent.setChildren(append(newChildren, oldChildren[idx:]...))
	return nil
}

func (t *treeNode) setChildren(ch []TreeNode) {
	t.children = ch
}

func (t *treeNode) NextSibling() TreeNode {
	if t.parent == nil {
		return nil
	}

	next := t.SiblingIndex() + 1
	siblings := t.parent.ChildNodes()
	if next < len(siblings) {
		return siblings[next]
	}
	return nil
}

func (t *treeNode) PreviousSibling() TreeNode {
	if t.parent == nil {
		return nil
	}

	siblings := t.parent.ChildNodes()
	if len(siblings) == 0 {
		return nil
	}

	prev := t.SiblingIndex() - 1
	if prev < 0 {
		return nil
	}
	return siblings[prev]
}

func (t *treeNode) ChildNodes() []TreeNode {
	ch := make([]TreeNode, len(t.children))
	copy(ch, t.children)
	return ch
}

func (t *treeNode) RemoveChildAt(idx int) TreeNode {
	child := t.ChildAt(idx)
	if child == nil {
		return nil
	}

	if len(t.children) > idx {
		t.children = append(t.children[:idx], t.children[idx+1:]...)
	} else {
		t.children = t.children[:idx]
	}
	return child
}

func (t *treeNode) InsertChildAt(idx int, child TreeNode) {
	if idx < 0 {
		panic(fmt.Sprintf("cannot insert child to index: %d", idx))
	}

	child.SetParent(t)
	if idx >= len(t.children) {
		t.AppendChild(child)
		return
	}

	t.children = append(t.children[:idx],
		append([]TreeNode{child}, t.children[idx:]...)...)
}

func (t *treeNode) RemoveChild(child TreeNode) {
	idx := t.childIndex(child)
	if idx < 0 {
		return
	}
	t.RemoveChildAt(idx)
}

func (t *treeNode) AppendChild(child TreeNode) {
	child.SetParent(t)
	t.children = append(t.children, child)
}

func (t *treeNode) ChildAt(idx int) TreeNode {
	if idx > -1 && len(t.children) > idx {
		return t.children[idx]
	}
	return nil
}

func (t *treeNode) childIndex(child TreeNode) int {
	for i, c := range t.children {
		if c == child {
			return i
		}
	}
	return -1
}
