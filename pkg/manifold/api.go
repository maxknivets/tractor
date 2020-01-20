package manifold

import (
	"reflect"
)

// TreeNode is an interface for managing an object in a
// tree data structure.
type TreeNode interface {

	// Root returns the top-most parent to this node
	// if there are any parents, otherwise it returns nil.
	Root() Object

	// Parent returns the parent node to this node
	// if there is one, otherwise it returns nil.
	Parent() Object

	// SetParent moves this node under the given node.
	// note: triggers a change for objects
	SetParent(node Object)

	// SiblingIndex returns the order of this node relative
	// to its sibling nodes under their parent. If there is
	// no parent it will return 0.
	SiblingIndex() int

	// SetSiblingIndex changes the position of this node
	// relative to its sibling nodes. If the node has no
	// parent or the index is out of range, it will return
	// an error. Using an index of -1 will set it to the
	// highest possible index.
	// note: triggers a change for objects
	SetSiblingIndex(idx int) error

	// NextSibling returns the next the sibling of this node
	// or nil if there is no parent.
	NextSibling() Object

	// PreviousSibling returns the previous sibling of this node
	// or nil if there is no parent.
	PreviousSibling() Object

	// Children returns a slice of any child nodes of this node.
	Children() []Object

	// RemoveChildAt removes and return the child node at the given index.
	// note: triggers a change for objects
	RemoveChildAt(idx int) Object

	// InsertChildAt inserts the node as a child at the given index.
	// note: triggers a change for objects
	InsertChildAt(idx int, node Object)

	// RemoveChild removes the given node if it is a child of this node.
	// note: triggers a change for objects
	RemoveChild(node Object)

	// AppendChild adds the given node to this node's children.
	// note: triggers a change for objects
	AppendChild(child Object)

	// ChildAt returns the child node at the given index. Using an
	// index of -1 will return the last child.
	ChildAt(idx int) Object
}

type ComponentList interface {

	// AppendComponent adds a component to the component list.
	// note: triggers a change for objects
	AppendComponent(com Component)

	// RemoveComponent removes the given component from the component list.
	// note: triggers a change for objects
	RemoveComponent(com Component)

	// InsertComponentAt inserts a component to the component list at the index
	// specified.
	// note: triggers a change for objects
	InsertComponentAt(idx int, com Component)

	// RemoveComponentAt removes and returns the component at the given index.
	// note: triggers a change for objects
	RemoveComponentAt(idx int) Component

	// HasComponent checks if a component is already in the list.
	HasComponent(com Component) bool

	// Component returns the component with the given name from the component list.
	// Since the component name is typically a type and there can be several components
	// of the same type, the name for each includes a suffix `/` and its index among
	// the others of the same type. Example: `http.Server/0` returns the first component
	// named `http.Server`.
	Component(name string) Component

	// Components returns all the components in the component list.
	Components() []Component
}

// AttributeSet is an interface for managing internal key-value
// attributes of an object.
// note: potentially implemented on a map[string]interface{} instead of a struct
type AttributeSet interface {
	// HasAttribute returns true if the named attribute exists.
	HasAttribute(attr string) bool

	// GetAttribute returns the named attribute value.
	GetAttribute(attr string) interface{}

	// SetAttribute sets the named attribute value.
	// note: triggers a change for objects
	SetAttribute(attr string, value interface{})

	// UnsetAttribute removes the named attribute.
	// note: triggers a change for objects
	UnsetAttribute(attr string)
}

type ComponentGetter interface {
	// GetField returns the field value found at the given
	// path relative to the current context.
	GetField(path string) (interface{}, error)
}

type ComponentSetter interface {
	// SetField sets the field value at the given path relative
	// to the current context.
	// note: triggers a change for objects
	SetField(path string, value interface{}) error
}

type ComponentCaller interface {
	// CallMethod will call the method at the given path relative
	// to the current context.
	CallMethod(path string, args []interface{}, reply interface{}) error
}

// Component is an interface for working with components meant to
// be contained by an object. They often wrap a pointer to a struct
// but can also represent a "virtual" component.
// note: should be Component, but name exists in old implementation atm
type Component interface {
	ComponentGetter
	ComponentSetter
	ComponentCaller

	// Index returns the position among other components
	// on the containing object. It always returns 0 if there is no
	// containing object.
	Index() int

	// SetIndex sets the position among other components
	// on the containing object. Using an index of -1 will set it to the
	// highest possible index.
	// note: triggers a change for objects
	SetIndex(idx int)

	// ID returns a unique identifier for this object.
	// ID() string

	// Name returns the name of the component.
	Name() string

	// Enabled returns whether the component is enabled.
	// Specifically what that means is TBD...
	Enabled() bool

	// SetEnabled sets whether the component is enabled.
	// note: triggers a change for objects
	SetEnabled(enable bool)

	// Container returns the containing object of this component.
	Container() Object

	// SetContainer sets the containing object of this component.
	// note: triggers a change for objects
	SetContainer(obj Object)

	// Pointer returns a reference to the struct value behind this
	// component or nil if there is none.
	Pointer() interface{}

	// Snapshot returns a marshalable version of the data behind
	// this component, such as a version of the Pointer value with
	// pointer fields replaced with ID references.
	Snapshot() ComponentSnapshot

	// Type returns a reflect.Type for the value of Pointer if there
	// is one.
	Type() reflect.Type

	FieldType(path string) reflect.Type

	// TODO
	Fields()

	// TODO
	Methods()

	// TODO
	RelatedComponents()

	// TODO
	RelatedPrefabs()
}

type ObjectObserver struct {
	// Path is a prefix to the paths that change notifications
	// are allowed to come from for this observer. If blank, it
	// defaults to the path of the object.
	Path string

	// OnChange is a callback for change notifications. It gets the
	// specific object that was changed, the path to the specific
	// field that was changed, the old value, and the new value.
	OnChange func(obj Object, path string, old, new interface{})
}

// Object is the main primitive of Tractor, which is made up of components
// and child objects. They can either be part of a workspace System or a Prefab.
type Object interface {
	TreeNode
	ComponentList
	AttributeSet
	ComponentGetter
	ComponentSetter
	ComponentCaller

	// Name returns the name of this object.
	Name() string

	// Path returns the absolute path of this object.
	Path() string

	// Subpath returns a subpath of this object.
	Subpath(names ...string) string

	// SetName sets the name of this object.
	// note: triggers a change for this object
	SetName(name string)

	// ID returns a unique identifier for this object.
	ID() string

	// FindChild returns a descendant of this object that
	// macthes the name or relative path. It returns nil
	// if no descendant matches.
	FindChild(subpath string) Object

	// FindPointer returns a descendant of this object that
	// contains a component pointing to the given struct
	// reference. It returns nil if no components match.
	FindPointer(ptr interface{}) Object

	// ValueTo sets a component value from this objects
	// registry based on the type of reflect.Value
	ValueTo(rv reflect.Value)

	FindID(id string) Object

	RemoveID(id string) Object

	// Observe registers an observer with the object that
	// will be notified of changes to the object.
	Observe(observer *ObjectObserver)

	// Unobserve unregisters an observer with the object to
	// no longer be notified of changes.
	Unobserve(observer *ObjectObserver)

	// Main returns the main component for this object
	// or nil if there is none. The main component is not kept in
	// the ComponentList.
	Main() Component

	// SetMain sets the main component for this object.
	// If the component exists in the ComponentList, it will be removed.
	// note: triggers a change for this object
	SetMain(com Component)

	Snapshot() ObjectSnapshot

	UpdateRegistry() error
}
