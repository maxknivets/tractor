package http

import (
	"net/http"
	"reflect"

	"github.com/manifold/tractor/pkg/manifold"
)

type Mux struct {
	obj manifold.Object `hash:"ignore"`
}

func (c *Mux) InitializeComponent(obj manifold.Object) {
	c.obj = obj
}

func (c *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	for _, child := range c.obj.Children() {
		var handler http.Handler
		child.ValueTo(reflect.ValueOf(&handler))
		if handler != nil {
			mux.Handle("/"+child.Name()+"/", handler)
		}
	}
	mux.ServeHTTP(w, r)
}
