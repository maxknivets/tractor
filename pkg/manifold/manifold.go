package manifold

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"strings"

	jsonpointer "github.com/dustin/go-jsonpointer"
	"github.com/gliderlabs/com/objects"

	"github.com/mitchellh/hashstructure"
	"github.com/mitchellh/mapstructure"
	reflected "github.com/progrium/prototypes/go-reflected"
	"github.com/rs/xid"
)

var (
	registeredDelegates  map[string]reflected.Type
	registeredComponents []reflected.Type
	componentFilepaths   map[string]string
)

var (
	// TODO: this isn't necessary
	Root *Node

	// TODO: ugh. find a better way. until then, maybe put a lock on LoadHierarchy
	DeflatedRefs []DeflatedRef
)

func init() {
	Root = NewNode("")

	registeredDelegates = make(map[string]reflected.Type)
	componentFilepaths = make(map[string]string)
}

func RegisterDelegate(v interface{}, id string) {
	registeredDelegates[id] = reflected.ValueOf(v).Type()
}

func RegisterComponent(v interface{}, path string) {
	if path == "" {
		_, path, _, _ = runtime.Caller(1)
	}
	ctype := reflected.ValueOf(v).Type()
	registeredComponents = append(registeredComponents, ctype)
	componentFilepaths[ctype.Name()] = path
}

func RegisteredComponents() []string {
	var names []string
	for _, typ := range registeredComponents {
		names = append(names, typ.Name())
	}
	return names
}

func RegisteredComponentPath(n string) string {
	return componentFilepaths[n]
}

func RegisteredDelegates() []string {
	var ids []string
	for k, _ := range registeredDelegates {
		ids = append(ids, k)
	}
	return ids
}

func NewComponent(name string) interface{} {
	for _, typ := range registeredComponents {
		if typ.Name() == name {
			return reflected.New(typ).Interface()
		}
	}
	return nil
}

func NewDelegate(id string) interface{} {
	typ, ok := registeredDelegates[id]
	if ok {
		return reflected.New(typ).Interface()
	}
	return nil
}

type DeflatedRef struct {
	NodeID     string
	Path       string
	TargetID   string
	TargetType reflect.Type
}

type Node struct {
	Children []*Node
	Active   bool
	Name     string
	ID       string
	Dir      string `json:"-"`

	Components []*Component

	observers  map[*NodeObserver]struct{}
	parent     *Node
	lastActive bool
	lastName   string
	Registry   *objects.Registry `json:"-"`
}

func Walk(n *Node, fn func(*Node)) {
	if n.parent != nil {
		fn(n)
	}
	for _, child := range n.Children {
		Walk(child, fn)
	}
}

type NodeObserver struct {
	Path     string
	OnChange func(node *Node, path string, old, new interface{})
}

func NewNode(name string) *Node {
	n := &Node{
		Name:   name,
		Active: true,
		ID:     xid.New().String(),
	}
	n.TempInflate()
	return n
}

func (n *Node) SetDelegate(v interface{}) {
	if n.Delegate() != nil {
		return
	}
	com := componentFromValue(v, n)
	com.Delegate = true
	n.Components = append([]*Component{com}, n.Components[:]...)
	n.Registry.Register(objects.New(com, ""))
	n.Sync()
}

func (n *Node) FullPath() string {
	p := []string{}
	if n.parent != nil {
		p = append(p, n.Name)
	}
	parent := n.parent
	for parent != nil {
		if parent.parent != nil {
			p = append([]string{parent.Name}, p...)
		}
		parent = parent.parent
	}
	p = append([]string{""}, p...)
	return strings.Join(p, "/")
}

func (n *Node) ExpandPath(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	if strings.HasPrefix(path, "../") && n.parent != nil {
		return n.parent.ExpandPath(path[3:])
	}
	return n.FullPath() + "/" + path
}

func (n *Node) ValuePaths() []string {
	var s []string
	for _, com := range n.Components {
		ptrs, _ := jsonpointer.ReflectListPointers(com.Ref)
		for _, p := range ptrs {
			s = append(s, com.Name+p)
		}
	}
	return s
}

func (n *Node) CallMethod(localPath string) {
	com, methodName := splitComponentPath(localPath)
	rcom := reflect.ValueOf(n.Component(com))
	method := rcom.MethodByName(methodName[1:])
	method.Call(nil)
}

func splitComponentPath(path string) (string, string) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return path, ""
	}
	return parts[0], "/" + parts[1]
}

func (n *Node) Value(path string) interface{} {
	com, valPath := splitComponentPath(path)
	return jsonpointer.Reflect(n.Component(com), valPath)
}

func (n *Node) Expression(path string) string {
	com, valPath := splitComponentPath(path)
	for _, c := range n.Components {
		if c.Expr == nil {
			c.Expr = make(map[string]string)
		}
		if c.Name == com {
			return c.Expr[valPath]
		}
	}
	return ""
}

func (n *Node) SetExpression(path, value string) {
	com, valPath := splitComponentPath(path)
	for _, c := range n.Components {
		if c.Name == com {
			if c.Expr == nil {
				c.Expr = make(map[string]string)
			}
			c.Expr[valPath] = value
			// p := n.ExpandPath(value)
			// subject := n.FindNode(p)
			// if subject == nil {
			// 	panic("can't find " + p)
			// }
			// localPath := p[len(subject.FullPath())+1:]
			// subject.Observe(&NodeObserver{
			// 	Path: localPath,
			// 	OnChange: func(_ *Node, p string, old, new interface{}) {
			// 		n.evaluateExpression(path)
			// 	},
			// })
			n.evaluateExpression(path)
			return
		}
	}
}

func (n *Node) evaluateExpression(localPath string) {
	// expr := n.Expression(localPath)
	// var ret interface{}
	// repl := repl.NewREPL(func(v interface{}) {
	// 	ret = v
	// })
	// in := bytes.NewBufferString(expr + "\n")
	// repl.Run(in, ioutil.Discard, map[string]interface{}{
	// 	"Node": n,
	// })
	// if ret != nil {
	// 	n.SetValue(localPath, ret)
	// }

	// referencedNode := n.FindNode(referencePath)
	// refLocalPath := referencePath[len(referencedNode.FullPath()):]
	// v := referencedNode.Value(refLocalPath[1:])

	// parent := jsonpointer.Reflect(n.Component(com), parentPath)
	// field := reflect.ValueOf(&parent).Elem().Field(0)
	// field.Set(reflect.ValueOf(v))
	// rparent := reflected.ValueOf(parent)
	// rparent.Set(path.Base(localPath), v)
}

func (n *Node) SetValue(localPath string, v interface{}) {
	com, valuePath := splitComponentPath(localPath)
	SetReflect(n.Component(com), valuePath, v)
	n.Sync()
}

func (n *Node) Field(localPath string) reflect.Type {
	com, fieldPath := splitComponentPath(localPath)
	comType := reflected.TypeOf(n.Component(com))
	// TODO: subfields...
	fieldPath = strings.Replace(fieldPath, "/", "", -1)
	return comType.FieldType(fieldPath).Type
}

func (n *Node) Observe(observer *NodeObserver) {
	if n.observers == nil {
		n.observers = make(map[*NodeObserver]struct{})
	}
	n.observers[observer] = struct{}{}
}

func (n *Node) Unobserve(observer *NodeObserver) {
	if n.observers == nil {
		return
	}
	delete(n.observers, observer)
}

// func (n *Node) FindValue(path string) interface{} {

// }

func (n *Node) FindNode(path string) *Node {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == "" && len(parts) > 1 {
		//fmt.Println(strings.Join(parts[1:], "/"))
		return n.Root().FindNode(strings.Join(parts[1:], "/"))
	}
	if parts[0] == ".." {
		if n.parent == nil {
			return nil
		}
		if len(parts) == 1 {
			return n.parent
		}
		return n.parent.FindNode(strings.Join(parts[1:], "/"))
	}
	if n.Component(parts[0]) != nil {
		return n
	}
	child := n.Child(parts[0])
	if child == nil {
		return nil
	}
	if len(parts) == 1 {
		return child
	}
	return child.FindNode(strings.Join(parts[1:], "/"))
}

func (n *Node) Root() *Node {
	if n.parent == nil {
		return n
	}
	return n.parent.Root()
}

func (n *Node) Child(name string) *Node {
	for _, child := range n.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func (n *Node) FindPtr(v interface{}) *Node {
	for _, com := range n.Components {
		if com.Ref == v {
			return n
		}
	}
	for _, child := range n.Children {
		if res := child.FindPtr(v); res != nil {
			return res
		}
	}
	return nil
}

func (n *Node) FindID(id string) *Node {
	if n.ID == id {
		return n
	}
	for _, child := range n.Children {
		if child.ID == id {
			return child
		}
	}
	for _, child := range n.Children {
		if res := child.FindID(id); res != nil {
			return res
		}
	}
	return nil
}

func (n *Node) SiblingIndex() int {
	if n.parent == nil {
		return 0
	}
	for idx, child := range n.parent.Children {
		if child == n {
			return idx
		}
	}
	return 0
}

func (n *Node) SetSiblingIndex(idx int) {
	if n.parent == nil {
		return
	}
	if idx < 0 {
		return
	}
	if idx >= len(n.parent.Children) {
		return
	}
	oldIndex := n.SiblingIndex()
	oldChildren := n.parent.Children
	oldChildren = append(oldChildren[:oldIndex], oldChildren[oldIndex+1:]...)
	newChildren := make([]*Node, idx+1)
	copy(newChildren, oldChildren[:idx])
	newChildren[idx] = n
	newChildren = append(newChildren, oldChildren[idx:]...)
	n.parent.Children = newChildren
	n.parent.notifyObservers(n.parent, n.Name, nil, nil) // is this right?
}

func (n *Node) RemoveAt(idx int) *Node {
	node := n.Children[idx]
	n.Children = append(n.Children[:idx], n.Children[idx+1:]...)
	// n.Sync()
	n.notifyObservers(n, node.Name, nil, nil)
	return node
}

func (n *Node) RemoveID(id string) *Node {
	for idx, child := range n.Children {
		if child.ID == id {
			return n.RemoveAt(idx)
		}
	}
	return nil
}

func (n *Node) Remove() {
	n.parent.RemoveID(n.ID)
}

func (n *Node) Insert(idx int, node *Node) {
	node.parent = n
	n.Children = append(n.Children[:idx], append([]*Node{node}, n.Children[idx:]...)...)
}

func (n *Node) Append(node *Node) {
	node.parent = n
	n.Children = append(n.Children, node)
	// n.Sync()
	n.notifyObservers(n, node.Name, nil, nil)
}

func (n *Node) Component(name string) interface{} {
	for _, com := range n.Components {
		if com.Name == name {
			return com.Ref
		}
	}
	return nil
}

func (n *Node) RemoveComponentAt(idx int) interface{} {
	v := n.Components[idx]
	n.Components = append(n.Components[:idx], n.Components[idx+1:]...)
	n.Sync()
	n.notifyObservers(n, v.Name, nil, nil)
	return v.Ref
}

func (n *Node) RemoveComponent(name string) interface{} {
	for idx, com := range n.Components {
		if com.Name == name {
			return n.RemoveComponentAt(idx)
		}
	}
	return nil
}

func (n *Node) InsertComponent(idx int, v interface{}) {
	com := componentFromValue(v, n)
	n.Components = append(n.Components[:idx], append([]*Component{com}, n.Components[idx:]...)...)
	n.Registry.Register(objects.New(v, ""))
	n.Sync()
}

func (n *Node) AppendComponent(v interface{}) {
	com := componentFromValue(v, n)
	n.Components = append(n.Components, com)
	n.Registry.Register(objects.New(v, ""))
	n.Sync()
}

func (n *Node) Sync() error {
	if err := n.Registry.Reload(); err != nil {
		return err
	}
	if n.lastName != n.Name {
		n.notifyObservers(n, "Name", n.lastName, n.Name)
		n.lastName = n.Name
	}
	if n.lastActive != n.Active {
		n.notifyObservers(n, "Active", n.lastActive, n.Active)
		n.lastActive = n.Active
	}
	for _, com := range n.Components {
		hash, err := hashstructure.Hash(com.Ref, nil)
		if err != nil {
			return err
		}
		if com.lastValues == nil {
			com.lastValues = make(map[string]interface{})
		}
		if com.lastHash != hash {
			ptrs, _ := jsonpointer.ReflectListPointers(com.Ref)
			for _, ptr := range ptrs {
				if ptr == "" {
					continue
				}
				v := jsonpointer.Reflect(com.Ref, ptr)
				if reflect.ValueOf(v).Kind() == reflect.Struct {
					continue
				}
				if reflect.ValueOf(v).Kind() == reflect.Map {
					continue
				}
				if reflect.ValueOf(v).Kind() == reflect.Slice {
					continue
				}
				path := com.Name + ptr
				if com.lastValues[ptr] != v {
					n.notifyObservers(n, path, com.lastValues[ptr], v)
					com.lastValues[ptr] = v
				}
			}
			com.lastHash = hash
		}
	}
	return nil
}

func (n *Node) notifyObservers(node *Node, path string, old, new interface{}) {
	for observer, _ := range n.observers {
		if strings.HasPrefix(path, observer.Path) {
			observer.OnChange(node, path, old, new)
		}
	}
	if n.parent != nil {
		n.parent.notifyObservers(node, path, old, new)
	}
}

// TODO: rethink this
// NOTE: this is because you can't use registry to create references to Nodes
type ComponentInitializer interface {
	InitializeComponent(n *Node)
}

func (n *Node) TempInflate() error {
	n.Registry = &objects.Registry{}
	n.Registry.Register(objects.New(n, "node"))
	if n.Delegate() == nil {
		d := NewDelegate(n.ID)
		if d != nil {
			n.SetDelegate(d)
		}
	}
	for _, com := range n.Components {
		if com.node == nil {
			com.node = n
		}
		n.Registry.Register(objects.New(com.Ref, ""))
		initializer, ok := com.Ref.(ComponentInitializer)
		if ok {
			initializer.InitializeComponent(n)
		}
	}
	n.Sync()
	return nil
}

func (n *Node) Delegate() interface{} {
	for _, com := range n.Components {
		if com.Delegate {
			return com.Ref
		}
	}
	return nil
}

func componentFromValue(v interface{}, n *Node) *Component {
	return &Component{
		Name:   reflected.ValueOf(v).Type().Name(),
		Ref:    v,
		Expr:   make(map[string]string),
		NodeID: n.ID,
		node:   n,
	}
}

type Component struct {
	Name     string
	Ref      interface{}
	Expr     map[string]string
	NodeID   string // because unmarshalling is stateless and we need this
	Delegate bool

	node       *Node
	lastHash   uint64
	lastValues map[string]interface{}
}

type componentData struct {
	Name     string
	Delegate bool
	NodeID   string
	Data     json.RawMessage
}

func (c *Component) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"Name":     c.Name,
		"Delegate": c.Delegate,
		"NodeID":   c.NodeID,
		"Data":     deflatePointerReferences(c.Ref, c.node),
	}
	return json.Marshal(m)
}

func (c *Component) UnmarshalJSON(b []byte) error {
	var cd componentData
	err := json.Unmarshal(b, &cd)
	if err != nil {
		return err
	}
	c.Name = cd.Name
	c.Delegate = cd.Delegate
	c.NodeID = cd.NodeID
	var com interface{}
	if cd.Delegate {
		com = NewDelegate(cd.NodeID)
		if com == nil {
			return fmt.Errorf("delegate not found for '%s'", c.NodeID)
		}
	} else {
		com = NewComponent(c.Name)
		if com == nil {
			return fmt.Errorf("component '%s' not found", c.Name)
		}
	}
	var data map[string]interface{}
	err = json.Unmarshal(cd.Data, &data)
	if err != nil {
		return err
	}
	err = mapstructure.Decode(collectPointerReferences(com, data, c.Name, c.NodeID), com)
	if err != nil {
		return err
	}
	c.Ref = com
	return nil
}

func collectPointerReferences(obj, data interface{}, basePath, nodeID string) map[string]interface{} {
	out := make(map[string]interface{})
	robj := reflected.ValueOf(obj)
	rdata := reflected.ValueOf(data)
	rt := robj.Type()
	for _, field := range rt.Fields() {
		ft := rt.FieldType(field)
		fieldPath := path.Join(basePath, field)
		switch ft.Kind() {
		case reflect.Interface:
			if !rdata.HasKey(field) {
				continue
			}
			fv := rdata.Get(field)
			if fv.HasKey("$ref") && fv.Get("$ref").IsValid() {
				DeflatedRefs = append(DeflatedRefs, DeflatedRef{
					NodeID:     nodeID,
					Path:       fieldPath,
					TargetID:   fv.Get("$ref").String(),
					TargetType: ft.Type,
				})
				// node := root.FindID(fv.Get("$ref").String())
				// if node != nil {
				// 	ptr := reflect.New(ft)
				// 	node.Registry.ValueTo(ptr)
				// 	out[field] = reflect.Indirect(ptr).Interface()
				// 	log.Println(out)
				// }
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
			out[field] = collectPointerReferences(robj.Get(field).Interface(), rdata.Get(field).Interface(), fieldPath, nodeID)
		default:
			// TODO: slices need to be inflated too??
			if rdata.HasKey(field) {
				out[field] = rdata.Get(field).Interface()
			}
		}
	}
	return out
}

func deflatePointerReferences(obj interface{}, n *Node) map[string]interface{} {
	out := make(map[string]interface{})
	rv := reflected.ValueOf(obj)
	rt := rv.Type()
	for _, field := range rt.Fields() {
		ft := rt.FieldType(field)
		switch ft.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice:
			out[field] = deflatePointerReferences(rv.Get(field).Interface(), n)
		case reflect.Ptr, reflect.Interface:
			if rv.Get(field).IsNil() {
				continue
			}
			node := n.Root().FindPtr(rv.Get(field).Interface())
			if node == nil {
				out[field] = map[string]interface{}{"$ref": nil}
			} else {
				out[field] = map[string]interface{}{"$ref": node.ID}
			}
		default:
			out[field] = rv.Get(field).Interface()
		}
	}
	return out
}
