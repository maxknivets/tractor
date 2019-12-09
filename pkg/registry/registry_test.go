package registry

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type namedStruct struct {
	Name string
}

type number int

func (n number) String() string {
	return strconv.Itoa(int(n))
}

type fooString struct {
	Foo string
}

func (f *fooString) String() string {
	return f.Foo
}

func TestRegistry(t *testing.T) {
	v := namedStruct{
		Name: "bar",
	}
	e := Ref(&v)
	r := New()
	require.Nil(t, r.Register(e))

	entries := r.Entries()
	assert.Equal(t, 1, len(entries))

	vv := reflect.New(e.Type)
	r.ValueTo(vv)
	assert.Equal(t, v.Name, reflect.Indirect(vv).Interface().(namedStruct).Name)
}

func TestAssignableTo(t *testing.T) {
	v1 := &namedStruct{
		Name: "v1",
	}
	v2 := &namedStruct{
		Name: "v2",
	}
	r := New()
	require.Nil(t, r.Register(
		Ref(v1),
		Ref(v2),
	))

	typ := reflect.TypeOf(v1)
	entries := r.AssignableTo(typ)
	assert.Equal(t, 2, len(entries))
}

type injectTest struct {
	Foo    *namedStruct
	Foos   []*namedStruct
	Number fmt.Stringer
	unfoo  *namedStruct
}

func TestPopulate(t *testing.T) {
	r := New()
	require.Nil(t, r.Register(
		Ref(&fooString{"123"}),
		Ref(&namedStruct{
			Name: "foo1",
		}),
		Ref(&namedStruct{
			Name: "foo2",
		}),
	))

	obj := &injectTest{}
	r.Populate(obj)

	require.NotNil(t, obj.Foo)
	require.NotNil(t, obj.Number)
	require.Nil(t, obj.unfoo)
	assert.Equal(t, obj.Foo.Name, "foo1")
	assert.Equal(t, len(obj.Foos), 2)
	assert.Equal(t, obj.Number.String(), "123")
}
