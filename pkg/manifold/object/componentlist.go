package object

import (
	"strings"

	"github.com/manifold/tractor/pkg/manifold"
)

type componentlist struct {
	components []manifold.Component
}

func (l *componentlist) Components() []manifold.Component {
	c := make([]manifold.Component, len(l.components))
	copy(c, l.components)
	return c
}

func (l *componentlist) AppendComponent(com manifold.Component) {
	l.components = append(l.components, com)
}

func (l *componentlist) RemoveComponent(com manifold.Component) {
	defer com.SetContainer(nil)
	for idx, c := range l.components {
		if c == com {
			l.RemoveComponentAt(idx)
			return
		}
	}
}

func (l *componentlist) InsertComponentAt(idx int, com manifold.Component) {
	l.components = append(l.components[:idx], append([]manifold.Component{com}, l.components[idx:]...)...)
}

func (l *componentlist) RemoveComponentAt(idx int) manifold.Component {
	c := l.components[idx]
	l.components = append(l.components[:idx], l.components[idx+1:]...)
	c.SetContainer(nil)
	return c
}

func (l *componentlist) HasComponent(com manifold.Component) bool {
	for _, c := range l.components {
		if c == com {
			return true
		}
	}
	return false
}

func (l *componentlist) Component(name string) manifold.Component {
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
