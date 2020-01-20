package manifold

import (
	"reflect"

	reflected "github.com/progrium/prototypes/go-reflected"
)

func ExpandPath(o Object, path string) string {
	obj := o.FindChild(path)
	if obj == nil {
		return path
	}
	return obj.Path()
}

func Walk(o Object, fn func(Object)) {
	if o.Parent() != nil {
		fn(o)
	}
	for _, child := range o.Children() {
		Walk(child, fn)
	}
}

func DeflateReferences(root Object, v interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	rv := reflected.ValueOf(v)
	rt := rv.Type()
	for _, field := range rt.Fields() {
		ft := rt.FieldType(field)
		switch ft.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice:
			out[field] = DeflateReferences(root, rv.Get(field).Interface())
		case reflect.Ptr, reflect.Interface:
			if rv.Get(field).IsNil() {
				continue
			}
			if root == nil {
				out[field] = DeflateReferences(root, rv.Get(field).Interface())
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
