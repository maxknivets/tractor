package file

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/manifold/tractor/pkg/manifold"
	frontend "github.com/manifold/tractor/pkg/workspace/view"
)

func init() {
	manifold.RegisterComponent(&Local{}, "")
}

type Local struct {
	filepath string
	node     *manifold.Node `hash:"ignore"`
}

func (c *Local) InitializeComponent(n *manifold.Node) {
	c.node = n
}

func (c *Local) String() string {
	d, err := ioutil.ReadFile(c.filepath)
	if err != nil {
		log.Fatal(err)
	}
	return string(d)
}

func (c *Local) Initialize() {
	c.filepath = filepath.Join(c.node.Dir, "localFile")
	_, err := os.Stat(c.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Create(c.filepath)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (c *Local) InspectorButtons() []frontend.Button {
	return []frontend.Button{{
		Name:    "Edit File...",
		OnClick: fmt.Sprintf("window.vscode.postMessage({event: 'edit', Filepath: '%s'})", c.filepath),
	}}
}
