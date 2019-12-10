package registry

import (
	"errors"
	"reflect"
	"sync"
)

// Entry is a reference to a value and reflected type data for that value.
type Entry struct {
	Ref      interface{}
	TypeName string
	PkgPath  string

	RefType reflect.Type
	Type    reflect.Type
	Value   reflect.Value
}

// Registry is a registry of value references that can be used to populate references
// to those values in other structs by type and interface.
type Registry struct {
	entries []*Entry

	mu sync.Mutex
}

// New returns a Registry optionally populated with entries for the given values.
func New(v ...interface{}) (*Registry, error) {
	r := &Registry{}
	return r, r.Register(v...)
}

// Entries returns the entry structs in the registry.
func (r *Registry) Entries() []*Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := make([]*Entry, len(r.entries))
	copy(e, r.entries)
	return e
}

// Register adds value pointers to the registry. Arguments can be an Entry or
// any other value, which will be wrapped in an Entry.
func (r *Registry) Register(v ...interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, vv := range v {
		var e *Entry
		switch ev := vv.(type) {
		case Entry:
			e = &ev
		case *Entry:
			e = ev
		default:
			// TODO: allow values instead of assuming pointer references
			if reflect.TypeOf(vv).Kind() != reflect.Ptr {
				vv = &vv
			}
			e = &Entry{Ref: vv}
		}
		e.RefType = reflect.TypeOf(e.Ref)

		// if not a pointer, ignore
		if e.RefType.Kind() != reflect.Ptr {
			return errors.New("value reference must be a pointer")
		}

		e.Type = e.RefType.Elem()
		e.Value = reflect.ValueOf(e.Ref)
		e.PkgPath = e.Type.PkgPath()
		e.TypeName = e.Type.Name()

		// error if the object has no package path
		// if e.TypeName == "" && e.PkgPath == "" {
		// 	return errors.New("unable to register object without name when it has no package path")
		// }

		// append entry to registry list
		r.entries = append(r.entries, e)

	}
	return nil
}

// AssignableTo returns entries that can be assigned to a value of the provided type.
func (r *Registry) AssignableTo(t reflect.Type) []*Entry {
	var entries []*Entry
	if t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	for _, entry := range r.Entries() {
		if entry.RefType.AssignableTo(t) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Populate will set any fields on the given struct that match a type or interface in the registry.
// It only sets fields that are exported. If there are more than one matches in the registry, the
// first one is used. If the field is a slice, it will be populated with all the matches in the registry
// for that slice type.
func (r *Registry) Populate(v interface{}) {
	rv := reflect.ValueOf(v)
	// TODO: assert struct
	var fields []reflect.Value
	for i := 0; i < rv.Elem().NumField(); i++ {
		// filter out unexported fields
		if len(rv.Elem().Type().Field(i).PkgPath) > 0 {
			continue
		}
		fields = append(fields, rv.Elem().Field(i))
		// TODO: filtering with struct tags
	}
	for _, field := range fields {
		if !isNilOrZero(field, field.Type()) {
			continue
		}
		assignable := r.AssignableTo(field.Type())
		if len(assignable) == 0 {
			continue
		}
		if field.Type().Kind() == reflect.Slice {
			field.Set(reflect.MakeSlice(field.Type(), 0, len(assignable)))
			for _, entry := range assignable {
				field.Set(reflect.Append(field, entry.Value))
			}
		} else {
			field.Set(assignable[0].Value)
		}
	}
}

// SelfPopulate will run Populate on each Entry in the registry.
func (r *Registry) SelfPopulate() {
	for _, e := range r.Entries() {
		r.Populate(e.Ref)
	}
}

func isNilOrZero(v reflect.Value, t reflect.Type) bool {
	switch v.Kind() {
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(t).Interface())
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
}

// ValueTo will set a reflect.Value to the first entry that matches the type
// of the reflect.Value. Remember to use reflect.Indirect on rv after.
func (r *Registry) ValueTo(rv reflect.Value) {
	for _, e := range r.Entries() {
		if rv.Elem().Type().Kind() == reflect.Struct {
			if e.Value.Elem().Type().AssignableTo(rv.Elem().Type()) {
				rv.Elem().Set(e.Value.Elem())
				return
			}
		} else {
			if e.Value.Type().Implements(rv.Elem().Type()) {
				rv.Elem().Set(e.Value)
				return
			}
		}
	}

}
