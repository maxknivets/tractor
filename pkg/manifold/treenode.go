package manifold

type treeNode struct {
	object Object
	parent TreeNode
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
