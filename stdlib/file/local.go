package file

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/manifold/tractor/pkg/manifold"
	frontend "github.com/manifold/tractor/pkg/workspace/view"
)

type Local struct {
	filepath string
	obj      manifold.Object `hash:"ignore"`
}

func (c *Local) InitializeComponent(obj manifold.Object) {
	c.obj = obj
}

func (c *Local) String() string {
	d, err := ioutil.ReadFile(c.filepath)
	if err != nil {
		log.Fatal(err)
	}
	return string(d)
}

func (c *Local) Initialize() error {
	// TODO: pls rethink!
	c.filepath = "/tmp/localFile-" + c.obj.ID()
	_, err := os.Stat(c.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Create(c.filepath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Local) InspectorButtons() []frontend.Button {
	return []frontend.Button{{
		Name:    "Edit File...",
		OnClick: fmt.Sprintf("window.vscode.postMessage({event: 'edit', Filepath: '%s'})", c.filepath),
	}}
}
