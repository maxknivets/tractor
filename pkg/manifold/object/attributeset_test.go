package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttributeSet(t *testing.T) {
	sys := New("")
	sys.SetAttribute("test", "value")

	assert.True(t, sys.HasAttribute("test"))
	assert.Equal(t, "value", sys.GetAttribute("test").(string))

	sys.SetAttribute("test", 42)

	assert.True(t, sys.HasAttribute("test"))
	assert.Equal(t, 42, sys.GetAttribute("test").(int))

	sys.UnsetAttribute("test")

	assert.False(t, sys.HasAttribute("test"))
	assert.Nil(t, sys.GetAttribute("test"))
}
