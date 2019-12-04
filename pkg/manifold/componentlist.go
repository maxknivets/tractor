package manifold

type componentlist struct {
	components []Component2
}

func (l *componentlist) Components() []Component2 {
	return l.components
}
