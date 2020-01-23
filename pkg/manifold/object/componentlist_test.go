package object

import (
	"testing"

	"github.com/manifold/tractor/pkg/manifold/library"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentList(t *testing.T) {
	t.Run("AppendComponent", func(t *testing.T) {
		cl := &componentlist{}
		require.Empty(t, cl.Components())
		com := library.NewComponent("com", "com", "")
		cl.AppendComponent(com)
		assert.NotEmpty(t, cl.Components())
	})

	t.Run("RemoveComponent", func(t *testing.T) {
		cl := &componentlist{}
		com := library.NewComponent("com", "com", "")
		cl.AppendComponent(com)
		require.NotEmpty(t, cl.Components())
		cl.RemoveComponent(com)
		assert.Empty(t, cl.Components())
	})

	t.Run("RemoveComponentAt", func(t *testing.T) {
		cl := &componentlist{}
		com1 := library.NewComponent("com1", "com1", "")
		cl.AppendComponent(com1)
		com2 := library.NewComponent("com2", "com2", "")
		cl.AppendComponent(com2)
		require.Len(t, cl.Components(), 2)
		cl.RemoveComponentAt(1)
		assert.Len(t, cl.Components(), 1)
	})

	t.Run("RemoveComponentAt", func(t *testing.T) {
		cl := &componentlist{}
		com1 := library.NewComponent("com1", "com1", "")
		cl.AppendComponent(com1)
		require.Len(t, cl.Components(), 1)
		com2 := library.NewComponent("com2", "com2", "")
		cl.InsertComponentAt(0, com2)
		assert.Len(t, cl.Components(), 2)
		assert.Equal(t, com2, cl.Components()[0])
	})
}
