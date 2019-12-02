// WIP! New API
// methods noted as triggering a change only
// actually do so in their implementation on Object.
package manifold

import "reflect"

// Objector represents a type that is also an Object,
// allowing easier access to the Object interface.
type Objector interface {
	Object() Object
}

// TreeNode is an interface for managing a node in a
// tree data structure.
type TreeNode interface {
	Objector

	// Root returns the top-most parent to this node
	// if there are any parents, otherwise it returns nil.
	Root() TreeNode

	// Parent returns the parent node to this node
	// if there is one, otherwise it returns nil.
	Parent() TreeNode

	// SetParent moves this node under the given node.
	// note: triggers a change for objects
	SetParent(node TreeNode)

	// SiblingIndex returns the order of this node relative
	// to its sibling nodes under their parent. If there is
	// no parent it will return 0.
	//SiblingIndex() int

	// SetSiblingIndex changes the position of this node
	// relative to its sibling nodes. If the node has no
	// parent or the index is out of range, it will return
	// an error. Using an index of -1 will set it to the
	// highest possible index.
	// note: triggers a change for objects
	//SetSiblingIndex(idx int) error

	// NextSibling returns the next the sibling of this node
	// or nil if there is no parent.
	//NextSibling() TreeNode

	// PreviousSibling returns the prevopis the sibling of this node
	// or nil if there is no parent.
	//PreviousSibling() TreeNode

	// ChildNodes returns a slice of any child nodes of this node.
	ChildNodes() []TreeNode

	// RemoveChildAt removes and return the child node at the given index.
	// note: triggers a change for objects
	RemoveChildAt(idx int) TreeNode

	// InsertChildAt inserts the node as a child at the given index.
	// note: triggers a change for objects
	InsertChildAt(idx int, node TreeNode)

	// RemoveChild removes the given node if it is a child of this node.
	// note: triggers a change for objects
	RemoveChild(node TreeNode)

	// AppendChild adds the given node to this node's children.
	// note: triggers a change for objects
	AppendChild(child TreeNode)

	// ChildAt returns the child node at the given index. Using an
	// index of -1 will return the last child.
	ChildAt(idx int) TreeNode
}

type ComponentList interface {

	// AppendComponent adds a component to the component list.
	// note: triggers a change for objects
	//AppendComponent(com Component2)

	// RemoveComponent removes the given component from the component list.
	// note: triggers a change for objects
	//RemoveComponent(com Component2)

	// InsertComponentAt inserts a component to the component list at the index
	// specified.
	// note: triggers a change for objects
	//InsertComponentAt(idx int, com Component2)

	// RemoveComponentAt removes and returns the component at the given index.
	// note: triggers a change for objects
	//RemoveComponentAt(idx int) Component2

	// Component returns the component with the given name from the component list.
	// Since the component name is typically a type and there can be several components
	// of the same type, the name for each includes a suffix `/` and its index among
	// the others of the same type. Example: `http.Server/0` returns the first component
	// named `http.Server`.
	//Component(name string) Component2

	// Components returns all the components in the component list.
	Components() []Component2
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
	CallMethod(path string, args, reply interface{}) error
}

// Component is an interface for working with components meant to
// be contained by an object. They often wrap a pointer to a struct
// but can also represent a "virtual" component.
// note: should be Component, but name exists in old implementation atm
type Component2 interface {
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
	SetIndex(idx int) error

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

	// MarshalData returns a marshalable version of the data behind
	// this component, such as a version of the Pointer value with
	// pointer fields replaced with ID references.
	MarshalData() interface{}

	// Type returns a reflect.Type for the value of Pointer if there
	// is one.
	Type() reflect.Type

	// TODO
	Fields()

	// TODO
	Methods()

	// TODO
	RelatedComponents()

	// TODO
	RelatedPrefabs()
}

// UserComponent is a user-defined component whose source is stored
// the workspace.
type UserComponent interface {
	// TODO

	// -where the code is stored
	// -any initialization
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
	//ComponentGetter
	//ComponentSetter
	//ComponentCaller

	// Children returns the children of this object.
	// note: convenience wrapper for TreeNode.ChildNodes() as Objects
	//Children() []Object

	// Name returns the name of this object.
	Name() string

	// SetName sets the name of this object.
	// note: triggers a change for this object
	//SetName(name string)

	// ID returns a unique identifier for this object.
	//ID() string

	// FindChild returns a descendant of this object that
	// macthes the name or relative path. It returns nil
	// if no descendant matches.
	//FindChild(subpath string) Object

	// FindPointer returns a descendant of this object that
	// contains a component pointing to the given struct
	// reference. It returns nil if no components match.
	//FindPointer(ptr interface{}) Object

	// Observe registers an observer with the object that
	// will be notified of changes to the object.
	Observe(observer *ObjectObserver)

	// Unobserve unregisters an observer with the object to
	// no longer be notified of changes.
	Unobserve(observer *ObjectObserver)

	Notify(object Object, path string, old, new interface{})

	// MainComponent returns the main component for this object
	// or nil if there is none. The main component is not kept in
	// the ComponentList. The main component is often a UserComponent.
	//MainComponent() Component

	// SetMainComponent sets the main component for this object.
	// If the component exists in the ComponentList, it will be removed.
	// note: triggers a change for this object
	//SetMainComponent(com Component)

	// System returns the root parent node of this object as a System
	// if it is one, otherwise it returns nil.
	//System() System
}

// System is the root Object in a workspace.
type System interface {
	Object

	// AbsPath returns a full path from root for a given object.
	//AbsPath(obj Object) string

	// ExpandPath returns a normalized absolute path for a relative
	// path to the given object.
	//ExpandPath(obj Object, path string) string

	// FindID will find an object in the system that has the given
	// ID or returns nil.
	FindID(id string) Object

	// RemoveID will remove and return the object from its parent that has the
	// given ID, otherwise it returns nil.
	//RemoveID(id string) Object
}
