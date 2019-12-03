package manifold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeNodeSetParent(t *testing.T) {
	sys := New("sys")
	obj := New("obj")

	assert.Nil(t, obj.Parent())
	assert.Equal(t, obj, obj.Root().Object())
	assert.Equal(t, sys.Object(), sys.Root().Object())

	obj.SetParent(sys)
	assert.Equal(t, sys, obj.Parent())
	assert.NotEqual(t, obj, obj.Parent())
	assert.Equal(t, sys.Object(), obj.Root().Object())
	assert.Equal(t, sys.Object(), sys.Root().Object())
}

func TestTreeNodeSiblingIndex(t *testing.T) {
	// test when child nodes are addes as *object or *treeNode
	addChildNodes := map[string]func(o Object) TreeNode{
		"object":   rootChildAsObject,
		"treeNode": rootChildAsTreeNode,
	}

	// test when SetSibling() is called on *object or *treeNode
	testSetSibling := map[string]func(Object) (TreeNode, TreeNode, TreeNode){
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
}

func setSiblingOnTreeNode(root Object) (TreeNode, TreeNode, TreeNode) {
	siblings := root.ChildNodes()
	return siblings[0], siblings[1], siblings[2]
}

func setSiblingOnObject(root Object) (TreeNode, TreeNode, TreeNode) {
	siblings := root.ChildNodes()
	return siblings[0].Object(), siblings[1].Object(), siblings[2].Object()
}

var (
	rootChildAsObject = func(o Object) TreeNode {
		return o
	}

	rootChildAsTreeNode = func(o Object) TreeNode {
		return o.Root()
	}
)

func TestTreeNodeNextSibling(t *testing.T) {
	root, c1, c2, c3 := rootWithSiblings(nil)
	assert.Nil(t, root.NextSibling())
	assert.Nil(t, root.PreviousSibling())
	assert.Equal(t, c2, c1.NextSibling())
	assert.Nil(t, c1.PreviousSibling())
	assert.Equal(t, c3, c2.NextSibling())
	assert.Equal(t, c1, c2.PreviousSibling())
	assert.Nil(t, c3.NextSibling())
	assert.Equal(t, c2, c3.PreviousSibling())
}

func rootWithSiblings(fn func(Object) TreeNode) (System, Object, Object, Object) {
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
	return root.System(), c1, c2, c3
}

func TestTreeNodeChildAt(t *testing.T) {
	sys := New("sys")
	assert.Equal(t, 0, len(sys.ChildNodes()))
	assert.Nil(t, sys.ChildAt(0))
	assert.Nil(t, sys.ChildAt(100))
}

func TestTreeNodeAppendChild(t *testing.T) {
	sys := New("sys")
	n1 := New("n1")
	assert.Nil(t, n1.Parent())
	sys.AppendChild(n1)
	assert.Equal(t, sys.Object(), n1.Parent().Object())

	assert.Equal(t, []string{"n1"}, childNodeNames(sys))
}

func TestTreeNodeInsertChildAt(t *testing.T) {
	sys := New("sys")
	n2 := New("n2")
	assert.Nil(t, n2.Parent())
	sys.InsertChildAt(0, n2)
	assert.Equal(t, sys.Object(), n2.Parent().Object())
	assert.Equal(t, []string{"n2"}, childNodeNames(sys))

	n3 := New("n3")
	assert.Nil(t, n3.Parent())
	sys.InsertChildAt(1, n3)
	assert.Equal(t, sys.Object(), n3.Parent().Object())
	assert.Equal(t, []string{"n2", "n3"}, childNodeNames(sys))

	n4 := New("n4")
	assert.Nil(t, n4.Parent())
	sys.InsertChildAt(5, n4)
	assert.Equal(t, sys.Object(), n4.Parent().Object())
	assert.Equal(t, []string{"n2", "n3", "n4"}, childNodeNames(sys))

	n1 := New("n1")
	assert.Nil(t, n1.Parent())
	sys.InsertChildAt(0, n1)
	assert.Equal(t, sys.Object(), n1.Parent().Object())
	assert.Equal(t, []string{"n1", "n2", "n3", "n4"}, childNodeNames(sys))
}

func TestTreeNodeRemoveChild(t *testing.T) {
	sys := New("sys")
	sys.AppendChild(New("n1"))
	sys.AppendChild(New("n2"))
	sys.AppendChild(New("n3"))

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
}

func childNodeNames(t TreeNode) []string {
	childNodes := t.ChildNodes()
	names := make([]string, len(childNodes))
	for i, c := range childNodes {
		names[i] = c.Object().Name()
	}
	return names
}
