package manifold

func (o *object) FindID(id string) Object {
	return nil
}

func (o *object) System() System {
	if o.parent != nil {
		return o.Root().Object().System()
	}
	return o
}
