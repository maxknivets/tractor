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
	"github.com/spf13/afero"
)

const (
	// ObjectPathAttr = "image.objpath"
	ObjectDir  = "obj"
	ObjectFile = "object.json"

	PackageDir = "pkg"
)

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
	library.Register(&Main{}, "%s")
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
		imports = append(imports, fmt.Sprintf(` _ "workspace/pkg/obj/%s"`, info.Name()))
	}
	src := fmt.Sprintf("package obj\nimport (\n%s\n)\n", strings.Join(imports, "\n"))

	return afero.WriteFile(i.pkgFs, path.Join(ObjectDir, "import.go"), []byte(src), 0644)
}

func (i *Image) Load() (manifold.Object, error) {
	i.objFs = afero.NewBasePathFs(i.fs, ObjectDir)

	if ok, err := afero.Exists(i.objFs, ObjectFile); !ok || err != nil {
		r := manifold.New("::root")
		r.AppendChild(manifold.New("System"))
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
		ptr := reflect.New(ref.TargetType)
		dst.ValueTo(ptr)
		src.SetField(ref.Path, reflect.Indirect(ptr).Interface())
	}

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
	for _, com := range snapshot.Components {
		refs = append(refs, com.SnapshotRefs()...)
	}

	obj := manifold.FromSnapshot(snapshot)
	i.lastObjPath[obj.ID()] = path

	for _, childInfo := range snapshot.Children {
		childFs := afero.NewBasePathFs(fs, pathNameFromImage(childInfo))
		childPath := paths.Join(path, pathNameFromImage(childInfo))
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
