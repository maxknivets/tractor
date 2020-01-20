package manifold

import (
	"encoding/json"
	"path"
	"reflect"

	reflected "github.com/progrium/prototypes/go-reflected"
)

type ObjectSnapshot struct {
	ID         string
	Name       string
	Attrs      map[string]interface{}
	ParentID   string
	Children   [][]string
	Components []ComponentSnapshot
	Main       string
}

type ComponentSnapshot struct {
	ObjectID string
	ID       string
	Name     string
	Enabled  bool
	Attrs    map[string]interface{}
	Value    map[string]interface{}

	refs []SnapshotRef
}

func (c *ComponentSnapshot) UnmarshalJSON(b []byte) (err error) {
	err = json.Unmarshal(b, c)
	if err != nil {
		return
	}
	c.refs = scanRefs(c.Value, c.Name, c.ObjectID)
	return
}

func (c *ComponentSnapshot) SnapshotRefs() []SnapshotRef {
	return c.refs
}

type SnapshotRef struct {
	ObjectID   string
	Path       string
	TargetID   string
	TargetType reflect.Type
}

func scanRefs(v interface{}, basePath, objectID string) []SnapshotRef {
	var refs []SnapshotRef
	rv := reflected.ValueOf(v)
	rt := rv.Type()
	for _, field := range rt.Fields() {
		ft := rt.FieldType(field)
		fieldPath := path.Join(basePath, field)
		switch ft.Kind() {
		case reflect.Interface:
			fv := rv.Get(field)
			if fv.HasKey("$ref") && fv.Get("$ref").IsValid() {
				refs = append(refs, SnapshotRef{
					ObjectID:   objectID,
					Path:       fieldPath,
					TargetID:   fv.Get("$ref").String(),
					TargetType: ft.Type,
				})
			}
		case reflect.Struct, reflect.Map:
			// if ft.Kind() == reflect.Map {
			// 	if fv.HasKey("$ref") {
			// 		node := n.Root().FindID(fv.Get("$ref").String())
			// 		if node != nil {
			// 			comPtr := NewComponent(fv.Get("$type").String())
			// 			node.Registry.ValueTo(&comPtr)
			// 			out[field] = comPtr
			// 		}
			// 		continue
			// 	}
			// }
			refs = append(refs, scanRefs(rv.Get(field).Interface(), fieldPath, objectID)...)
		default:
			// TODO: slices need to be inflated too??
		}
	}
	return refs
}
