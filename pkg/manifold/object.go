package manifold

import (
	"errors"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/manifold/tractor/pkg/misc/debouncer"
	"github.com/manifold/tractor/pkg/misc/registry"
	"github.com/rs/xid"
)

func newObject(name string) *object {
	obj := &object{
		id:           xid.New().String(),
		name:         name,
		path:         "/",
		observers:    make(map[*ObjectObserver]struct{}),
		attributeset: attributeset(make(map[string]interface{})),
		componentlist: componentlist{
			components: make([]Component, 0),
		},
		notifyDebounce: debouncer.New(1000 * time.Millisecond),
	}
	return obj
}

func fromSnapshot(snapshot ObjectSnapshot) *object {
	obj := newObject(snapshot.Name)
	obj.id = snapshot.ID
	obj.attributeset = attributeset(snapshot.Attrs)
	for _, comSnap := range snapshot.Components {
		com := newComponent(comSnap.Name, comSnap.Value)
		com.enabled = comSnap.Enabled
		com.id = comSnap.ID
		obj.components = append(obj.components, com)
		if snapshot.Main != "" && comSnap.Name == snapshot.Main {
			obj.main = com
		}
	}
	return obj
}

func FromSnapshot(snapshot ObjectSnapshot) Object {
	return fromSnapshot(snapshot)
}

func New(name string) Object {
	return newObject(name)
}

type object struct {
	componentlist
	attributeset

	parent   Object
	children []Object

	id        string
	name      string
	path      string
	observers map[*ObjectObserver]struct{}
	main      Component
	registry  *registry.Registry
	mu        sync.Mutex

	notifyDebounce func(f func())
	observerMu     sync.Mutex
}

func (o *object) GetField(path string) (interface{}, error) {
	parts := strings.SplitN(path, "/", 2)
	com := o.Component(parts[0])
	if com == nil {
		return nil, errors.New("component not on node: " + parts[0])
	}
	return com.GetField(parts[1])
}

func (o *object) SetField(path string, value interface{}) error {
	parts := strings.SplitN(path, "/", 2)
	com := o.Component(parts[0])
	if com == nil {
		return errors.New("component not on node: " + parts[0])
	}
	return com.SetField(parts[1], value)
}

func (o *object) CallMethod(path string, args []interface{}, reply interface{}) error {
	parts := strings.SplitN(path, "/", 2)
	com := o.Component(parts[0])
	if com == nil {
		return errors.New("component not on node: " + parts[0])
	}
	return com.CallMethod(parts[1], args, reply)
}

func (o *object) ValueTo(rv reflect.Value) {
	o.registry.ValueTo(rv)
}

func (o *object) Name() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.name
}

func (o *object) SetName(name string) {
	old := o.Name()
	if old != name {
		o.mu.Lock()
		o.name = name
		o.mu.Unlock()
		o.notify(o, "::Name", old, name)
	}
}

func (o *object) ID() string {
	return o.id
}

func (o *object) Path() string {
	// o.mu.Lock()
	// defer o.mu.Unlock()
	parts := []string{}
	var obj Object = o
	for obj.Parent() != nil {
		parts = append([]string{obj.Name()}, parts...)
		obj = obj.Parent()
	}
	return "/" + path.Join(parts...)
}

func (o *object) Subpath(names ...string) string {
	parts := []string{o.Path()}
	return path.Join(append(parts, names...)...)
}

func (o *object) FindChild(subpath string) Object {
	parts := strings.Split(subpath, "/")
	if len(parts) == 1 {
		if parts[0] == "." {
			return o
		}
		if parts[0] == ".." {
			if o.Parent() == nil {
				return nil
			}
			return o.Parent()
		}
		for _, child := range o.Children() {
			if child.Name() == parts[0] {
				return child
			}
		}
		return nil
	}
	var obj Object = o
	for _, part := range parts {
		obj = obj.FindChild(part)
		if obj == nil {
			return nil
		}
	}
	return obj
}

func (o *object) FindPointer(ptr interface{}) Object {
	for _, com := range o.Components() {
		if com.Pointer() == ptr {
			return o
		}
	}
	for _, child := range o.Children() {
		if obj := child.FindPointer(ptr); obj != nil {
			return obj
		}
	}
	return nil
}

func (o *object) Observe(obs *ObjectObserver) {
	o.observerMu.Lock()
	defer o.observerMu.Unlock()
	o.observers[obs] = struct{}{}
}

func (o *object) Unobserve(obs *ObjectObserver) {
	o.observerMu.Lock()
	defer o.observerMu.Unlock()
	delete(o.observers, obs)
}

func (o *object) Main() Component {
	return o.main
}

func (o *object) SetMain(com Component) {
	if !o.HasComponent(com) {
		o.InsertComponentAt(0, com)
	}
	old := o.main
	if old != com {
		o.main = com
		o.notify(o, "::Main", old, com)
	}
}

func (o *object) FindID(id string) Object {
	return findChildID(o, id)
}

func findChildID(p Object, id string) Object {
	for _, child := range p.Children() {
		if child.ID() == id {
			return child
		}
	}
	for _, child := range p.Children() {
		if obj := findChildID(child, id); obj != nil {
			return obj
		}
	}
	return nil
}

func (o *object) RemoveID(id string) Object {
	obj := o.FindID(id)
	if obj == nil {
		return nil
	}
	obj.Parent().RemoveChild(obj)
	return obj
}

// TODO: rethink this
// NOTE: this is because you can't use registry to create references to Nodes
type ComponentInitializer interface {
	InitializeComponent(o Object)
}

func (o *object) UpdateRegistry() (err error) {
	entries := []interface{}{Object(o)}
	for _, com := range o.Components() {
		entries = append(entries, com.Pointer())
		initializer, ok := com.Pointer().(ComponentInitializer)
		if ok {
			initializer.InitializeComponent(o)
		}
	}
	o.registry, err = registry.New(entries...)
	return
}

func (o *object) Snapshot() ObjectSnapshot {
	obj := ObjectSnapshot{
		ID:    o.ID(),
		Name:  o.Name(),
		Attrs: o.attributeset,
	}
	if o.Parent() != nil {
		obj.ParentID = o.Parent().ID()
	}
	if o.Main() != nil {
		obj.Main = o.Main().Name()
	}
	for _, child := range o.Children() {
		obj.Children = append(obj.Children, []string{child.ID(), child.Name()})
	}
	for _, com := range o.Components() {
		obj.Components = append(obj.Components, com.Snapshot())
	}
	return obj
}
