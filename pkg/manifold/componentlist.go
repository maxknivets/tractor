package manifold

import "strings"

type componentlist struct {
	components []Component
}

func (l *componentlist) Components() []Component {
	c := make([]Component, len(l.components))
	copy(c, l.components)
	return c
}

func (l *componentlist) AppendComponent(com Component) {
	l.components = append(l.components, com)
}

func (l *componentlist) RemoveComponent(com Component) {
	defer com.SetContainer(nil)
	for idx, c := range l.components {
		if c == com {
			l.RemoveComponentAt(idx)
			return
		}
	}
}

func (l *componentlist) InsertComponentAt(idx int, com Component) {
	l.components = append(l.components[:idx], append([]Component{com}, l.components[idx:]...)...)
}

func (l *componentlist) RemoveComponentAt(idx int) Component {
	c := l.components[idx]
	l.components = append(l.components[:idx], l.components[idx+1:]...)
	c.SetContainer(nil)
	return c
}

func (l *componentlist) HasComponent(com Component) bool {
	for _, c := range l.components {
		if c == com {
			return true
		}
	}
	return false
}

func (l *componentlist) Component(name string) Component {
	// support taking a relative path for convenience
	path := strings.Split(name, "/")
	name = path[0]
	for _, c := range l.components {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
