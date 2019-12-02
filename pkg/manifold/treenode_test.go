package manifold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParent(t *testing.T) {
	t.Run("sets parent", func(t *testing.T) {
		obj := NewObject("obj")
		p1 := NewObject("p1")

		assert.Nil(t, obj.Parent())
		assert.Equal(t, obj, obj.Root().Object())
		assert.Equal(t, p1, p1.Root().Object())

		obj.SetParent(p1)
		assert.Equal(t, p1, obj.Parent())
		assert.NotEqual(t, obj, obj.Parent())
		assert.Equal(t, p1, obj.Root().Object())
		assert.Equal(t, p1, p1.Root().Object())
	})
}
