package manifold

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testComponent struct {
	Foo string
}

func (c *testComponent) Echo(args ...string) []string {
	return args
}

func (c *testComponent) Err(msg string) (interface{}, error) {
	return nil, errors.New(msg)
}

func TestComponent(t *testing.T) {
	t.Run("GetSetFields", func(t *testing.T) {
		obj := &testComponent{Foo: "foo"}
		com := newComponent("test", obj)
		v, _ := com.GetField("Foo")
		assert.Equal(t, "foo", v)
		com.SetField("Foo", "bar")
		assert.Equal(t, "bar", obj.Foo)
	})
	t.Run("CallMethod", func(t *testing.T) {
		obj := &testComponent{Foo: "foo"}
		com := newComponent("test", obj)

		var echoRet []string
		noerr := com.CallMethod("Echo", []interface{}{"1", "2", "3"}, &echoRet)
		assert.Nil(t, noerr)
		assert.Len(t, echoRet, 3)

		err := com.CallMethod("Err", []interface{}{"error"}, nil)
		assert.Error(t, err)
		assert.Equal(t, "error", err.Error())
	})
}
