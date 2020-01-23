package object

import (
	"fmt"
	"testing"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeNode(t *testing.T) {
	t.Run("SetParent", func(t *testing.T) {
		sys := New("sys")
		obj := New("obj")

		assert.Nil(t, obj.Parent())
		assert.Equal(t, obj, obj.Root())
		assert.Equal(t, sys, sys.Root())

		obj.SetParent(sys)
		assert.Equal(t, sys, obj.Parent())
		assert.NotEqual(t, obj, obj.Parent())
		assert.Equal(t, sys, obj.Root())
		assert.Equal(t, sys, sys.Root())
	})

	t.Run("SiblingIndex", func(t *testing.T) {
		// test when child nodes are addes as *object or *treeNode
		addChildNodes := map[string]func(o manifold.Object) manifold.Object{
			"object":   rootChildAsObject,
			"treeNode": rootChildAsTreeNode,
		}

		// test when SetSibling() is called on *object or *treeNode
		testSetSibling := map[string]func(manifold.Object) (manifold.Object, manifold.Object, manifold.Object){
			"object":   setSiblingOnObject,
			"treeNode": setSiblingOnTreeNode,
		}

		for desc, rootFn := range addChildNodes {
			for desc2, testFn := range testSetSibling {
				t.Run(fmt.Sprintf("%s.AppendChild/%s.SetSibling", desc, desc2), func(t *testing.T) {
					root, s1, s2, s3 := rootWithSiblings(rootFn)
					assert.Equal(t, 0, root.SiblingIndex())

					assert.Equal(t, []string{"c1", "c2", "c3"}, childNodeNames(root))
					assert.Equal(t, 0, s1.SiblingIndex())
					assert.Equal(t, 1, s2.SiblingIndex())
					assert.Equal(t, 2, s3.SiblingIndex())

					c1, c2, c3 := testFn(root)
					assert.Nil(t, c2.SetSiblingIndex(2))

					assert.Equal(t, []string{"c1", "c3", "c2"}, childNodeNames(root))
					assert.Equal(t, 0, c1.SiblingIndex())
					assert.Equal(t, 2, c2.SiblingIndex())
					assert.Equal(t, 1, c3.SiblingIndex())

					assert.Nil(t, c3.SetSiblingIndex(0))
					assert.Equal(t, []string{"c3", "c1", "c2"}, childNodeNames(root))
					assert.Equal(t, 1, c1.SiblingIndex())
					assert.Equal(t, 2, c2.SiblingIndex())
					assert.Equal(t, 0, c3.SiblingIndex())

					assert.NotNil(t, c3.SetSiblingIndex(-1))
					assert.NotNil(t, c3.SetSiblingIndex(3))
				})
			}
		}
	})

	t.Run("NextSibling", func(t *testing.T) {
		root, c1, c2, c3 := rootWithSiblings(nil)
		assert.Nil(t, root.NextSibling())
		assert.Nil(t, root.PreviousSibling())
		assert.Equal(t, c2, c1.NextSibling())
		assert.Nil(t, c1.PreviousSibling())
		assert.Equal(t, c3, c2.NextSibling())
		assert.Equal(t, c1, c2.PreviousSibling())
		assert.Nil(t, c3.NextSibling())
		assert.Equal(t, c2, c3.PreviousSibling())
	})

	t.Run("ChildAt", func(t *testing.T) {
		sys := New("sys")
		assert.Equal(t, 0, len(sys.Children()))
		assert.Nil(t, sys.ChildAt(0))
		assert.Nil(t, sys.ChildAt(100))
	})

	t.Run("AppendChild", func(t *testing.T) {
		sys := New("sys")
		n1 := New("n1")
		assert.Nil(t, n1.Parent())
		sys.AppendChild(n1)
		assert.Equal(t, sys, n1.Parent())

		assert.Equal(t, []string{"n1"}, childNodeNames(sys))
	})

	t.Run("InsertChildAt", func(t *testing.T) {
		sys := New("sys")
		n2 := New("n2")
		assert.Nil(t, n2.Parent())
		sys.InsertChildAt(0, n2)
		assert.Equal(t, sys, n2.Parent())
		assert.Equal(t, []string{"n2"}, childNodeNames(sys))

		n3 := New("n3")
		assert.Nil(t, n3.Parent())
		sys.InsertChildAt(1, n3)
		assert.Equal(t, sys, n3.Parent())
		assert.Equal(t, []string{"n2", "n3"}, childNodeNames(sys))

		n4 := New("n4")
		assert.Nil(t, n4.Parent())
		sys.InsertChildAt(5, n4)
		assert.Equal(t, sys, n4.Parent())
		assert.Equal(t, []string{"n2", "n3", "n4"}, childNodeNames(sys))

		n1 := New("n1")
		assert.Nil(t, n1.Parent())
		sys.InsertChildAt(0, n1)
		assert.Equal(t, sys, n1.Parent())
		assert.Equal(t, []string{"n1", "n2", "n3", "n4"}, childNodeNames(sys))
	})

	t.Run("RemoveChild", func(t *testing.T) {
		sys := New("sys")
		sys.AppendChild(New("n1"))
		sys.AppendChild(New("n2"))
		sys.AppendChild(New("n3"))

		require.Equal(t, []string{"n1", "n2", "n3"}, childNodeNames(sys))
		n1 := sys.Children()[0]
		n2 := sys.Children()[1]
		n3 := sys.Children()[2]

		sys.RemoveChild(n2)
		require.Equal(t, []string{"n1", "n3"}, childNodeNames(sys))
		sys.RemoveChild(n1)
		require.Equal(t, []string{"n3"}, childNodeNames(sys))
		sys.RemoveChild(n3)
		require.Equal(t, []string{}, childNodeNames(sys))
	})

	t.Run("RemoveChildAt", func(t *testing.T) {
		sys := New("sys")
		sys.AppendChild(New("n1"))
		sys.AppendChild(New("n2"))
		sys.AppendChild(New("n3"))

		require.Equal(t, []string{"n1", "n2", "n3"}, childNodeNames(sys))

		sys.RemoveChildAt(1)
		require.Equal(t, []string{"n1", "n3"}, childNodeNames(sys))
		sys.RemoveChildAt(0)
		require.Equal(t, []string{"n3"}, childNodeNames(sys))
		sys.RemoveChildAt(0)
		require.Equal(t, []string{}, childNodeNames(sys))
	})

}

func setSiblingOnTreeNode(root manifold.Object) (manifold.Object, manifold.Object, manifold.Object) {
	siblings := root.Children()
	return siblings[0], siblings[1], siblings[2]
}

func setSiblingOnObject(root manifold.Object) (manifold.Object, manifold.Object, manifold.Object) {
	siblings := root.Children()
	return siblings[0], siblings[1], siblings[2]
}

var (
	rootChildAsObject = func(o manifold.Object) manifold.Object {
		return o
	}

	rootChildAsTreeNode = func(o manifold.Object) manifold.Object {
		return o.Root()
	}
)

func rootWithSiblings(fn func(manifold.Object) manifold.Object) (manifold.Object, manifold.Object, manifold.Object, manifold.Object) {
	root := New("root")
	c1 := New("c1")
	c2 := New("c2")
	c3 := New("c3")
	if fn == nil {
		fn = rootChildAsObject
	}
	root.AppendChild(fn(c1))
	root.AppendChild(fn(c2))
	root.AppendChild(fn(c3))
	return root, c1, c2, c3
}

func childNodeNames(t manifold.TreeNode) []string {
	childNodes := t.Children()
	names := make([]string, len(childNodes))
	for i, c := range childNodes {
		names[i] = c.Name()
	}
	return names
}
