package manifold

type system struct {
	*object
}

func New() System {
	sys := &system{
		object: newObject(""),
	}
	sys.object.sys = sys
	return sys
}

func (s *system) FindID(id string) Object {
	return nil
}

func (s *system) System() System {
	return s
}
