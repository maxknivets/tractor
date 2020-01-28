package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/manifold/tractor/pkg/manifold"
	"github.com/manifold/tractor/pkg/manifold/library"
	"github.com/manifold/tractor/pkg/manifold/object"
	"github.com/radovskyb/watcher"
)

const WatchInterval = 100 * time.Millisecond

type FrontendUpdater interface {
	UpdateView()
}

type Explorer struct {
	Filepath string
	Object   manifold.Object
	System   FrontendUpdater

	Watcher        *watcher.Watcher
	parentExplorer *Explorer
	childExplorers map[string]*Explorer
}

func (c *Explorer) Watch(e *Explorer) error {
	if c.Watcher == nil {
		panic("watch called before watcher is set")
	}
	if c.parentExplorer == nil {
		c.childExplorers[e.Filepath] = e
		return c.Watcher.Add(e.Filepath)
	}
	return c.parentExplorer.Watch(e)
}

func (c *Explorer) Unwatch(e *Explorer) error {
	if c.Watcher == nil {
		return nil
	}
	if c.parentExplorer == nil {
		defer c.Watcher.Close()
		return c.Watcher.Remove(e.Filepath)
	}
	return c.parentExplorer.Unwatch(e)
}

func (c *Explorer) ComponentEnable() {
	c.childExplorers = make(map[string]*Explorer)
	if !c.exists() {
		return
	}

	parent := c.Object.Parent()
	if parent != nil {
		if com := parent.Component("Explorer"); com != nil {
			c.parentExplorer = com.Pointer().(*Explorer)
			c.Watcher = c.parentExplorer.Watcher
		}
	}

	if c.Watcher == nil {
		fmt.Println("NEW WATCHER")
		c.Watcher = watcher.New()
		c.Watcher.IgnoreHiddenFiles(true)
		c.Watcher.FilterOps(watcher.Create, watcher.Move, watcher.Rename, watcher.Remove)
	}
	if err := c.Watch(c); err != nil {
		panic(err)
	}
	if c.parentExplorer != nil {
		return
	}
	go func() {
		go c.handleChanges()
		if err := c.Watcher.Start(WatchInterval); err != nil {
			panic(err)
		}
	}()
}

func (c *Explorer) ComponentDisable() {
	c.Unwatch(c)
}

func (c *Explorer) handleChanges() {
	for {
		select {
		case event := <-c.Watcher.Event:
			fmt.Println(event)
			e, ok := c.childExplorers[filepath.Dir(event.Path)]
			if !ok {
				panic("no explorer for path " + event.Path)
			}
			switch event.Op {
			case watcher.Create:
				obj := object.New(event.Name())
				if event.IsDir() {
					obj.AppendComponent(library.NewComponent("Explorer", &Explorer{
						Filepath: event.Path,
					}, ""))
				}
				e.Object.AppendChild(obj)
			case watcher.Remove:
				obj := e.Object.FindChild(event.Name())
				if obj != nil {
					e.Object.RemoveChild(obj)
				}
			case watcher.Rename:
				obj := e.Object.FindChild(event.Name())
				if obj != nil {
					obj.SetName(filepath.Base(event.Path))
					if c := obj.Component("Explorer"); c != nil {
						c.Pointer().(*Explorer).Filepath = event.Path
					}
				}
			case watcher.Move:
				oe, ok := c.childExplorers[filepath.Dir(event.OldPath)]
				if !ok {
					panic("no explorer for path " + event.Path)
				}
				obj := oe.Object.FindChild(event.Name())
				if obj != nil {
					obj.SetName(filepath.Base(event.Path))
					if c := obj.Component("Explorer"); c != nil {
						c.Pointer().(*Explorer).Filepath = event.Path
					}
				}
				if oe != e {
					obj.SetParent(e.Object)
				}
			}
			c.System.UpdateView()
		case err := <-c.Watcher.Error:
			panic(err)
		case <-c.Watcher.Closed:
			return
		}
	}
}

func (c *Explorer) exists() bool {
	if c.Filepath == "" {
		return false
	}
	if _, err := os.Stat(c.Filepath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (c *Explorer) ChildNodes() (objs []manifold.Object) {
	if !c.exists() {
		return
	}
	fi, err := ioutil.ReadDir(c.Filepath)
	if err != nil {
		panic(err)
	}
	for _, f := range fi {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if f.Name() == "local" {
			continue
		}
		if f.Name() == "node_modules" {
			continue
		}
		obj := object.New(f.Name())
		if f.IsDir() {
			obj.AppendComponent(library.NewComponent("Explorer", &Explorer{
				Filepath: path.Join(c.Filepath, f.Name()),
			}, ""))
		}
		objs = append(objs, obj)
	}
	return
}
