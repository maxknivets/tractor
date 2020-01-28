package image

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	paths "path"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/manifold/library"
	"github.com/manifold/tractor/pkg/manifold/object"
	"github.com/spf13/afero"
)

const (
	ObjectDir  = "obj"
	ObjectFile = "object.json"

	PackageDir = "pkg"
)

type componentInitializer interface {
	InitializeComponent(o manifold.Object)
}

type componentEnabler interface {
	ComponentEnable()
}
type componentDisabler interface {
	ComponentDisable()
}

type Image struct {
	fs       afero.Fs
	objFs    afero.Fs
	pkgFs    afero.Fs
	filepath string

	lastObjPath map[string]string
	writeMu     sync.Mutex
}

func New(filepath string) *Image {
	return &Image{
		filepath:    filepath,
		fs:          afero.NewBasePathFs(afero.NewOsFs(), filepath),
		lastObjPath: make(map[string]string),
	}
}

func (i *Image) DestroyObjectPackage(obj manifold.Object) error {
	i.pkgFs = afero.NewBasePathFs(i.fs, PackageDir)
	if err := i.pkgFs.RemoveAll(path.Join(ObjectDir, obj.ID())); err != nil {
		return err
	}
	return i.IndexObjectPackages()
}

func (i *Image) CreateObjectPackage(obj manifold.Object) error {
	i.pkgFs = afero.NewBasePathFs(i.fs, PackageDir)

	dir := path.Join(ObjectDir, obj.ID())
	if err := i.pkgFs.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filepath := path.Join(dir, "component.go")
	if ok, _ := afero.Exists(i.pkgFs, filepath); ok {
		return nil
	}

	src := fmt.Sprintf(`package object

import "github.com/manifold/tractor/pkg/manifold/library"

func init() {
	library.Register(&Main{}, "%s", "")
}

type Main struct{}
`, obj.ID())

	if err := afero.WriteFile(i.pkgFs, filepath, []byte(src), 0644); err != nil {
		return err
	}

	return i.IndexObjectPackages()
}

func (i *Image) IndexObjectPackages() error {
	i.pkgFs = afero.NewBasePathFs(i.fs, PackageDir)

	if err := i.pkgFs.MkdirAll(ObjectDir, 0755); err != nil {
		return err
	}

	imports := []string{}
	fi, err := afero.ReadDir(i.pkgFs, ObjectDir)
	if err != nil {
		return err
	}
	for _, info := range fi {
		if info.IsDir() {
			imports = append(imports, fmt.Sprintf(` _ "workspace/pkg/obj/%s"`, info.Name()))
		}
	}
	src := fmt.Sprintf("package obj\nimport (\n%s\n)\n", strings.Join(imports, "\n"))

	return afero.WriteFile(i.pkgFs, path.Join(ObjectDir, "import.go"), []byte(src), 0644)
}

func (i *Image) Load() (manifold.Object, error) {
	i.objFs = afero.NewBasePathFs(i.fs, ObjectDir)

	if ok, err := afero.Exists(i.objFs, ObjectFile); !ok || err != nil {
		r := object.New("::root")
		r.AppendChild(object.New("System"))
		return r, nil
	}

	obj, refs, err := i.loadObject(i.objFs, "/")
	if err != nil {
		return nil, err
	}

	for _, ref := range refs {
		src := obj.FindID(ref.ObjectID)
		if src == nil {
			log.Printf("no object found for snapshot ref at %s", ref.ObjectID)
			continue
		}
		dst := obj.FindID(ref.TargetID)
		if dst == nil {
			log.Printf("no object found for snapshot ref target at %s", ref.TargetID)
			continue
		}
		_, targetType, _ := src.GetField(ref.Path)
		ptr := reflect.New(targetType)
		dst.ValueTo(ptr)
		src.SetField(ref.Path, reflect.Indirect(ptr).Interface())
	}

	manifold.Walk(obj, func(o manifold.Object) {
		o.UpdateRegistry()
		for _, c := range o.Components() {
			if e, ok := c.Pointer().(componentEnabler); ok {
				e.ComponentEnable()
			}
		}
	})

	return obj, nil
}

func (i *Image) loadObject(fs afero.Fs, path string) (manifold.Object, []manifold.SnapshotRef, error) {
	// TODO: Handle missing components?

	buf, err := afero.ReadFile(fs, ObjectFile)
	if err != nil {
		return nil, nil, err
	}

	var snapshot manifold.ObjectSnapshot
	err = json.Unmarshal(buf, &snapshot)
	if err != nil {
		return nil, nil, err
	}

	var refs []manifold.SnapshotRef
	obj := object.FromSnapshot(snapshot)
	i.lastObjPath[obj.ID()] = path
	for _, c := range snapshot.Components {
		refs = append(refs, c.Refs...)
		com := library.NewComponent(c.Name, c.Value, c.ID)
		com.SetEnabled(c.Enabled)
		obj.AppendComponent(com)
		if snapshot.Main != "" && c.ID == snapshot.Main {
			obj.SetMain(com)
		}
	}
	if snapshot.Main == "" && obj.Main() == nil {
		if com := library.LookupID(obj.ID()); com != nil {
			obj.SetMain(com.New())
		}
	}

	for _, childInfo := range snapshot.Children {
		name := pathNameFromImage(childInfo)
		if ok, err := afero.Exists(fs, name); !ok || err != nil {
			continue
		}
		childFs := afero.NewBasePathFs(fs, name)
		childPath := paths.Join(path, name)
		child, childRefs, err := i.loadObject(childFs, childPath)
		if err != nil {
			return nil, nil, err
		}
		obj.AppendChild(child)
		refs = append(refs, childRefs...)
	}

	return obj, refs, obj.UpdateRegistry()
}

func (i *Image) Write(root manifold.Object) error {
	i.writeMu.Lock()
	defer i.writeMu.Unlock()

	i.objFs = afero.NewBasePathFs(i.fs, ObjectDir)

	if err := i.fs.MkdirAll(ObjectDir, 0755); err != nil {
		return err
	}

	return i.writeObject(i.objFs, "/", root)
}

func (i *Image) writeObject(fs afero.Fs, path string, obj manifold.Object) error {
	i.lastObjPath[obj.ID()] = path
	iobj := obj.Snapshot()

	buf, err := json.MarshalIndent(iobj, "", "  ")
	if err != nil {
		return err
	}
	if err := afero.WriteFile(fs, ObjectFile, buf, 0644); err != nil {
		return err
	}

	for _, child := range obj.Children() {
		childPath := paths.Join(path, pathName(child))
		oldPath := i.lastObjPath[child.ID()]
		if oldPath != "" && oldPath != childPath {
			if err := i.objFs.Rename(oldPath, childPath); err != nil {
				return err
			}
		}
		if err := fs.MkdirAll(pathName(child), 0755); err != nil {
			return err
		}
		childFs := afero.NewBasePathFs(fs, pathName(child))
		if err := i.writeObject(childFs, childPath, child); err != nil {
			return err
		}
	}

	return nil
}

func pathNameFromImage(parts []string) string {
	shortid := parts[0][len(parts[0])-8:]
	exp := regexp.MustCompile("[^a-zA-Z0-9]+")
	name := strings.ToLower(exp.ReplaceAllString(parts[1], ""))
	return fmt.Sprintf("%s-%s", name[:min(8, len(name))], shortid)
}

func pathName(obj manifold.Object) string {
	return pathNameFromImage([]string{obj.ID(), obj.Name()})
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
