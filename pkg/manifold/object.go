package manifold

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

func (o *object) Name() string {
	return o.name
}

func (o *object) System() System {
	if o.sys != nil {
		return o.sys
	}
	return o.Root().Object().System()
}
