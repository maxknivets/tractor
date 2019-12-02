package manifold

type system struct {
	object
}

func New() System {
	return &system{
		object: *newObject(""),
	}
}

func (s *system) FindID(id string) Object {
	return nil
}
