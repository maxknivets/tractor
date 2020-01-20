package manifold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentList(t *testing.T) {
	t.Run("AppendComponent", func(t *testing.T) {
		cl := &componentlist{}
		require.Empty(t, cl.Components())
		com := newComponent("com", "com")
		cl.AppendComponent(com)
		assert.NotEmpty(t, cl.Components())
	})

	t.Run("RemoveComponent", func(t *testing.T) {
		cl := &componentlist{}
		com := newComponent("com", "com")
		cl.AppendComponent(com)
		require.NotEmpty(t, cl.Components())
		cl.RemoveComponent(com)
		assert.Empty(t, cl.Components())
	})

	t.Run("RemoveComponentAt", func(t *testing.T) {
		cl := &componentlist{}
		com1 := newComponent("com1", "com1")
		cl.AppendComponent(com1)
		com2 := newComponent("com2", "com2")
		cl.AppendComponent(com2)
		require.Len(t, cl.Components(), 2)
		cl.RemoveComponentAt(1)
		assert.Len(t, cl.Components(), 1)
	})

	t.Run("RemoveComponentAt", func(t *testing.T) {
		cl := &componentlist{}
		com1 := newComponent("com1", "com1")
		cl.AppendComponent(com1)
		require.Len(t, cl.Components(), 1)
		com2 := newComponent("com2", "com2")
		cl.InsertComponentAt(0, com2)
		assert.Len(t, cl.Components(), 2)
		assert.Equal(t, com2, cl.Components()[0])
	})
}
