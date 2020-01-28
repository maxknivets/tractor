package object

import (
	"errors"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/misc/debouncer"
	"github.com/manifold/tractor/pkg/misc/notify"
	"github.com/manifold/tractor/pkg/misc/registry"
	"github.com/rs/xid"
)

var RegistryPreloader func(o manifold.Object) []interface{}

func defaultPreloader(o manifold.Object) []interface{} {
	return []interface{}{o}
}

func init() {
	RegistryPreloader = defaultPreloader
}

func newObject(name string) *object {
	obj := &object{
		id:           xid.New().String(),
		name:         name,
		path:         "/",
		attributeset: attributeset(make(map[string]interface{})),
		componentlist: componentlist{
			components: make([]manifold.Component, 0),
		},
		notifyDebounce: debouncer.New(1000 * time.Millisecond),
	}
	return obj
}

func fromSnapshot(snapshot manifold.ObjectSnapshot) *object {
	obj := newObject(snapshot.Name)
	obj.id = snapshot.ID
	obj.attributeset = attributeset(snapshot.Attrs)
	return obj
}

func FromSnapshot(snapshot manifold.ObjectSnapshot) manifold.Object {
	return fromSnapshot(snapshot)
}

func New(name string) manifold.Object {
	return newObject(name)
}

type object struct {
	componentlist
	attributeset

	parent   manifold.Object
	children []manifold.Object

	id       string
	name     string
	path     string
	main     manifold.Component
	registry *registry.Registry
	mu       sync.Mutex

	notifyDebounce func(f func())
	t              notify.TopicImpl
}

func (o *object) GetField(path string) (interface{}, reflect.Type, error) {
	parts := strings.SplitN(path, "/", 2)
	com := o.Component(parts[0])
	if com == nil {
		return nil, nil, errors.New("component not on node: " + parts[0])
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
		// o.notify(o, "::Name", old, name)
		notify.Send(o, manifold.ObjectChange{
			Object: o,
			Path:   "::Name",
			Old:    old,
			New:    name,
		})
	}
}

func (o *object) ID() string {
	return o.id
}

func (o *object) Path() string {
	// o.mu.Lock()
	// defer o.mu.Unlock()
	parts := []string{}
	var obj manifold.Object = o
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

func (o *object) FindChild(subpath string) manifold.Object {
	parts := strings.Split(subpath, "/")
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == "" && len(parts) > 1 {
		return o.Root().FindChild(strings.Join(parts[1:], "/"))
	}
	if parts[0] == ".." {
		if o.parent == nil {
			return nil
		}
		if len(parts) == 1 {
			return o.parent
		}
		return o.parent.FindChild(strings.Join(parts[1:], "/"))
	}
	if o.Component(parts[0]) != nil {
		return o
	}
	var child manifold.Object
	for _, c := range o.Children() {
		if c.Name() == parts[0] {
			child = c
		}
	}
	if child == nil {
		return nil
	}
	if len(parts) == 1 {
		return child
	}
	return child.FindChild(strings.Join(parts[1:], "/"))
}

func (o *object) FindPointer(ptr interface{}) manifold.Object {
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

func (o *object) Observe(observer notify.Notifier) {
	o.t.Observe(observer)
}

func (o *object) Unobserve(observer notify.Notifier) {
	o.t.Unobserve(observer)
}

func (o *object) Notify(event interface{}) {
	o.t.Notify(event)
	notify.Send(o.parent, event)
}

func (o *object) Main() manifold.Component {
	return o.main
}

func (o *object) SetMain(com manifold.Component) {
	if !o.HasComponent(com) {
		o.InsertComponentAt(0, com)
	}
	old := o.main
	if old != com {
		o.main = com
		// o.notify(o, "::Main", old, com)
		notify.Send(o, manifold.ObjectChange{
			Object: o,
			Path:   "::Main",
			Old:    old,
			New:    com,
		})
	}
}

func (o *object) FindID(id string) manifold.Object {
	return findChildID(o, id)
}

func findChildID(p manifold.Object, id string) manifold.Object {
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

func (o *object) RemoveID(id string) manifold.Object {
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
	InitializeComponent(o manifold.Object)
}

func (o *object) UpdateRegistry() (err error) {
	entries := RegistryPreloader(o)
	for _, com := range o.Components() {
		entries = append(entries, com.Pointer())
	}
	o.registry, err = registry.New(entries...)
	return
}

func (o *object) Snapshot() manifold.ObjectSnapshot {
	obj := manifold.ObjectSnapshot{
		ID:    o.ID(),
		Name:  o.Name(),
		Attrs: o.attributeset,
	}
	if o.Parent() != nil {
		obj.ParentID = o.Parent().ID()
	}
	if o.Main() != nil {
		obj.Main = o.Main().ID()
	}
	for _, child := range o.Children() {
		obj.Children = append(obj.Children, []string{child.ID(), child.Name()})
	}
	for _, com := range o.Components() {
		obj.Components = append(obj.Components, com.Snapshot())
	}
	return obj
}
