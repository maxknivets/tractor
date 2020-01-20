package manifold

import (
	"fmt"
	"reflect"

	"github.com/manifold/tractor/pkg/misc/jsonpointer"
	reflected "github.com/progrium/prototypes/go-reflected"
)

type component struct {
	object  *object
	name    string
	id      string
	enabled bool
	value   interface{}
}

func newComponent(name string, value interface{}) *component {
	return &component{
		name:    name,
		enabled: true,
		value:   value,
	}
}

func NewComponent(name string, value interface{}) Component {
	return newComponent(name, value)
}

func (c *component) GetField(path string) (interface{}, error) {
	// TODO: check if field exists
	return jsonpointer.Reflect(c.value, path), nil
}

func (c *component) SetField(path string, value interface{}) error {
	old, _ := c.GetField(path)
	if old == value {
		return nil
	}
	jsonpointer.SetReflect(c.value, path, value)
	c.object.notify(c.object, fmt.Sprintf("%s/%s", c.name, path), old, value)
	return nil
}

func (c *component) FieldType(path string) reflect.Type {
	// TODO: subfields/subpath...
	rv := reflected.TypeOf(c.value)
	return rv.FieldType(path).Type
}

func (c *component) CallMethod(path string, args []interface{}, reply interface{}) error {
	// TODO: support methods on sub paths / data structures
	rval := reflect.ValueOf(c.value)
	method := rval.MethodByName(path)
	var params []reflect.Value
	for _, arg := range args {
		params = append(params, reflect.ValueOf(arg))
	}
	retVals := method.Call(params)
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	// assuming up to 2 return values, one being an error
	rreply := reflect.ValueOf(reply)
	var errVal error
	for _, v := range retVals {
		if v.Type().Implements(errorInterface) {
			if !v.IsNil() {
				errVal = v.Interface().(error)
			}
		} else {
			if reply != nil {
				rreply.Elem().Set(v)
			}
		}
	}
	return errVal
}

func (c *component) Index() int {
	for idx, com := range c.object.Components() {
		if com == c {
			return idx
		}
	}
	return 0
}

func (c *component) SetIndex(idx int) {
	if idx == -1 {
		idx = len(c.object.Components()) - 1
	}
	old := c.Index()
	if old == idx {
		return
	}
	c.object.RemoveComponent(c)
	c.object.InsertComponentAt(idx, c)
	c.object.notify(c.object, fmt.Sprintf("%s/::Index", c.name), old, idx)
}

func (c *component) Name() string {
	return c.name
}

func (c *component) ID() string {
	return c.id
}

func (c *component) Enabled() bool {
	return c.enabled
}

func (c *component) SetEnabled(enable bool) {
	old := c.enabled
	if old == enable {
		return
	}
	c.enabled = enable
	c.object.notify(c.object, fmt.Sprintf("%s/::Enabled", c.name), old, enable)
}

func (c *component) Container() Object {
	return c.object
}

func (c *component) SetContainer(obj Object) {
	if o, ok := obj.(*object); ok || obj == nil {
		c.object = o
	}
}

// TODO: rename to Value()?
func (c *component) Pointer() interface{} {
	return c.value
}

func (c *component) Type() reflect.Type {
	return reflect.TypeOf(c.value)
}

// TODO
func (c *component) Fields() {}

// TODO
func (c *component) Methods() {}

// TODO
func (c *component) RelatedComponents() {}

// TODO
func (c *component) RelatedPrefabs() {}

func (c *component) Snapshot() ComponentSnapshot {
	com := ComponentSnapshot{
		Name:    c.name,
		ID:      c.id,
		Enabled: c.enabled,
	}
	if c.object != nil {
		com.ObjectID = c.object.ID()
		com.Value = DeflateReferences(c.object.Root(), c.value)
	} else {
		com.Value = DeflateReferences(nil, c.value)
	}
	return com
}
