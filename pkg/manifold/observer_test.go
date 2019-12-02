package manifold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservers(t *testing.T) {
	t.Run("SetParent", func(t *testing.T) {
		seen := false

		obj := NewObject("test")
		parent := NewObject("parent")

		obj.Observe(&ObjectObserver{
			Path: "Parent",
			OnChange: func(changed Object, path string, old, new interface{}) {
				assert.Equal(t, obj, changed)
				assert.Equal(t, "Parent", path)
				assert.Equal(t, obj.Parent(), old)
				assert.Equal(t, parent, new)
				seen = true
			},
		})

		obj.Observe(&ObjectObserver{
			Path: "NotParent",
			OnChange: func(changed Object, path string, old, new interface{}) {
				t.Fatal("this observer has the wrong path")
			},
		})

		obj.SetParent(parent)
		require.True(t, seen)
	})
}
