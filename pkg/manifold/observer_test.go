package manifold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserveSetParent(t *testing.T) {
	seen := false
	obj := New("test")
	parent := New("parent")

	assertObserver(t, obj, "Parent", func(changed Object, path string, old, new interface{}) {
		assert.Equal(t, obj, changed)
		assert.Equal(t, "Parent", path)
		assert.Equal(t, obj.Parent(), old)
		assert.Equal(t, parent, new)
		seen = true
	})

	require.False(t, seen)
	obj.SetParent(parent)
	require.True(t, seen)
}

func TestObserveRemoveChildAt(t *testing.T) {
	seen := false
	obj := New("test")
	parent := New("parent")
	parent.AppendChild(obj)

	assertObserver(t, parent, "R", func(changed Object, path string, old, new interface{}) {
		assert.Equal(t, parent, changed)
		assert.Equal(t, "RemoveChildAt", path)
		assert.Equal(t, obj, old)
		assert.Nil(t, new)
		seen = true
	})

	require.False(t, seen)
	parent.RemoveChildAt(0)
	require.True(t, seen)
}

func TestObserveInsertChildAt(t *testing.T) {
	seen := false
	obj := New("test")
	parent := New("parent")

	assertObserver(t, parent, "I", func(changed Object, path string, old, new interface{}) {
		assert.Equal(t, parent, changed)
		assert.Equal(t, "InsertChildAt", path)
		assert.Nil(t, old)
		assert.Equal(t, obj, new)
		seen = true
	})

	require.False(t, seen)
	parent.InsertChildAt(0, obj)
	require.True(t, seen)
}

func TestObserveRemoveChild(t *testing.T) {
	seen := false
	obj := New("test")
	parent := New("parent")
	parent.AppendChild(obj)

	assertObserver(t, parent, "R", func(changed Object, path string, old, new interface{}) {
		assert.Equal(t, parent, changed)
		assert.Equal(t, "RemoveChild", path)
		assert.Equal(t, obj, old)
		assert.Nil(t, new)
		seen = true
	})

	require.False(t, seen)
	parent.RemoveChild(obj)
	require.True(t, seen)
}

func TestObserveAppendChild(t *testing.T) {
	seen := false
	obj := New("test")
	parent := New("parent")

	assertObserver(t, parent, "A", func(changed Object, path string, old, new interface{}) {
		assert.Equal(t, parent, changed)
		assert.Equal(t, "AppendChild", path)
		assert.Nil(t, old)
		assert.Equal(t, obj, new)
		seen = true
	})

	require.False(t, seen)
	parent.AppendChild(obj)
	require.True(t, seen)
}

func assertObserver(t *testing.T, obj Object, path string, onChange func(changed Object, path string, old, new interface{})) {
	obj.Observe(&ObjectObserver{
		Path: path,
		OnChange: func(changed Object, path string, old, new interface{}) {
			t.Logf("Observer OnChange %q %+v", path, changed)
			onChange(changed, path, old, new)
		},
	})

	refuteObserver(t, obj, "Not"+path)
}

func refuteObserver(t *testing.T, obj Object, path string) {
	obj.Observe(&ObjectObserver{
		Path: path,
		OnChange: func(changed Object, path string, old, new interface{}) {
			t.Fatal("this observer has the wrong path")
		},
	})
}
