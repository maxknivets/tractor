package manifold

func ExpandPath(o Object, path string) string {
	obj := o.FindChild(path)
	if obj == nil {
		return path
	}
	return obj.Path()
}

func Walk(o Object, fn func(Object)) {
	if o.Parent() != nil {
		fn(o)
	}
	for _, child := range o.Children() {
		Walk(child, fn)
	}
}
