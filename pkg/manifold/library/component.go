package library

import (
	"fmt"
	"reflect"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/misc/jsonpointer"
	"github.com/mitchellh/mapstructure"
	reflected "github.com/progrium/prototypes/go-reflected"
)

type component struct {
	object  manifold.Object
	name    string
	id      string
	enabled bool
	value   interface{}
}

func newComponent(name string, value interface{}, id string) *component {
	var typedValue interface{}
	if id == "" {
		// TODO: make up mind on how to use component IDs
		//id = xid.New().String()
		if rc := Lookup(name); rc != nil {
			typedValue = rc.NewValue()
		}
	} else {
		if rc := LookupID(id); rc != nil {
			typedValue = rc.NewValue()
		}
	}
	if typedValue != nil {
		if err := mapstructure.Decode(value, typedValue); err == nil {
			value = typedValue
		} else {
			panic(err)
		}
	}
	return &component{
		name:    name,
		enabled: true,
		value:   value,
		id:      id,
	}
}

func NewComponent(name string, value interface{}, id string) manifold.Component {
	return newComponent(name, value, id)
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
	// TODO: potentially replace this with an observer system that hooks in elsewhere
	if n, ok := c.object.(manifold.ObjectNotifier); ok {
		n.Notify(c.object, fmt.Sprintf("%s/%s", c.name, path), old, value)
	}
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
	if n, ok := c.object.(manifold.ObjectNotifier); ok {
		n.Notify(c.object, fmt.Sprintf("%s/::Index", c.name), old, idx)
	}
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
	if n, ok := c.object.(manifold.ObjectNotifier); ok {
		n.Notify(c.object, fmt.Sprintf("%s/::Enabled", c.name), old, enable)
	}
}

func (c *component) Container() manifold.Object {
	return c.object
}

func (c *component) SetContainer(obj manifold.Object) {
	c.object = obj
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

func (c *component) Snapshot() manifold.ComponentSnapshot {
	com := manifold.ComponentSnapshot{
		Name:    c.name,
		ID:      c.id,
		Enabled: c.enabled,
	}
	if c.object != nil {
		com.ObjectID = c.object.ID()
		com.Value = deflateReferences(c.object.Root(), c.value)
	} else {
		com.Value = deflateReferences(nil, c.value)
	}
	return com
}

func deflateReferences(root manifold.Object, v interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	rv := reflected.ValueOf(v)
	rt := rv.Type()
	for _, field := range rt.Fields() {
		ft := rt.FieldType(field)
		switch ft.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice:
			out[field] = deflateReferences(root, rv.Get(field).Interface())
		case reflect.Ptr, reflect.Interface:
			if rv.Get(field).IsNil() {
				continue
			}
			if root == nil {
				out[field] = deflateReferences(root, rv.Get(field).Interface())
			}
			obj := root.Root().FindPointer(rv.Get(field).Interface())
			if obj == nil {
				out[field] = map[string]interface{}{"$ref": nil}
			} else {
				out[field] = map[string]interface{}{"$ref": obj.ID()}
			}
		default:
			out[field] = rv.Get(field).Interface()
		}
	}
	return out
}
