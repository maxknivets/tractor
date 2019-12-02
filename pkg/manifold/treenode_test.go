package manifold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeNodeSetParent(t *testing.T) {
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
}

func TestTreeNodeChildAt(t *testing.T) {
	sys := New()
	assert.Equal(t, 0, len(sys.ChildNodes()))
	assert.Nil(t, sys.ChildAt(0))
	assert.Nil(t, sys.ChildAt(100))
}

func TestTreeNodeAppendChild(t *testing.T) {
	sys := New()
	n1 := NewObject("n1")
	assert.Nil(t, n1.Parent())
	sys.AppendChild(n1)
	assert.Equal(t, sys.Object(), n1.Parent().Object())

	assert.Equal(t, []string{"n1"}, childNodeNames(sys))
}

func TestTreeNodeInsertChildAt(t *testing.T) {
	sys := New()
	n2 := NewObject("n2")
	assert.Nil(t, n2.Parent())
	sys.InsertChildAt(0, n2)
	assert.Equal(t, sys.Object(), n2.Parent().Object())
	assert.Equal(t, []string{"n2"}, childNodeNames(sys))

	n3 := NewObject("n3")
	assert.Nil(t, n3.Parent())
	sys.InsertChildAt(1, n3)
	assert.Equal(t, sys.Object(), n3.Parent().Object())
	assert.Equal(t, []string{"n2", "n3"}, childNodeNames(sys))

	n4 := NewObject("n4")
	assert.Nil(t, n4.Parent())
	sys.InsertChildAt(5, n4)
	assert.Equal(t, sys.Object(), n4.Parent().Object())
	assert.Equal(t, []string{"n2", "n3", "n4"}, childNodeNames(sys))

	n1 := NewObject("n1")
	assert.Nil(t, n1.Parent())
	sys.InsertChildAt(0, n1)
	assert.Equal(t, sys.Object(), n1.Parent().Object())
	assert.Equal(t, []string{"n1", "n2", "n3", "n4"}, childNodeNames(sys))
}

func TestTreeNodeRemoveChild(t *testing.T) {
	sys := New()
	sys.AppendChild(NewObject("n1"))
	sys.AppendChild(NewObject("n2"))
	sys.AppendChild(NewObject("n3"))

	require.Equal(t, []string{"n1", "n2", "n3"}, childNodeNames(sys))
	n1 := sys.ChildNodes()[0]
	n2 := sys.ChildNodes()[1]
	n3 := sys.ChildNodes()[2]

	sys.RemoveChild(n2)
	require.Equal(t, []string{"n1", "n3"}, childNodeNames(sys))
	sys.RemoveChild(n1)
	require.Equal(t, []string{"n3"}, childNodeNames(sys))
	sys.RemoveChild(n3)
	require.Equal(t, []string{}, childNodeNames(sys))
}

func TestTreeNodeRemoveChildAt(t *testing.T) {
	sys := New()
	sys.AppendChild(NewObject("n1"))
	sys.AppendChild(NewObject("n2"))
	sys.AppendChild(NewObject("n3"))

	require.Equal(t, []string{"n1", "n2", "n3"}, childNodeNames(sys))

	sys.RemoveChildAt(1)
	require.Equal(t, []string{"n1", "n3"}, childNodeNames(sys))
	sys.RemoveChildAt(0)
	require.Equal(t, []string{"n3"}, childNodeNames(sys))
	sys.RemoveChildAt(0)
	require.Equal(t, []string{}, childNodeNames(sys))
}

func childNodeNames(t TreeNode) []string {
	childNodes := t.ChildNodes()
	names := make([]string, len(childNodes))
	for i, c := range childNodes {
		names[i] = c.Object().Name()
	}
	return names
}
