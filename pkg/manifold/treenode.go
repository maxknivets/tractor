package manifold

import "fmt"

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

func (t *treeNode) ChildNodes() []TreeNode {
	return t.children
}

func (t *treeNode) RemoveChildAt(idx int) TreeNode {
	child := t.ChildAt(idx)
	t.children = append(t.children[:idx], t.children[idx+1:]...)
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
