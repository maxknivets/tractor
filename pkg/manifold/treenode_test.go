package manifold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParent(t *testing.T) {
	t.Run("sets parent", func(t *testing.T) {
		sys := New()
		obj := NewObject("obj")

		assert.Nil(t, obj.Parent())
		assert.Equal(t, obj, obj.Root().Object())
		assert.Equal(t, sys.Object(), sys.Root().Object())

		obj.SetParent(sys)
		assert.Equal(t, sys, obj.Parent())
		assert.NotEqual(t, obj, obj.Parent())
		assert.Equal(t, sys.Object(), obj.Root().Object())
		assert.Equal(t, sys.Object(), sys.Root().Object())
	})
}
