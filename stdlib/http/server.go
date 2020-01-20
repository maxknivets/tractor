package http

import (
	"log"
	"net"
	"net/http"

	"github.com/manifold/tractor/pkg/workspace/view"
	"github.com/urfave/negroni"
)

type Server struct {
	Listener net.Listener `com:"singleton"`
	Handler  http.Handler `com:"singleton"`
	// Middleware []negroni.Handler `com:"extpoint"`

	s *http.Server
}

func (c *Server) InspectorButtons() []view.Button {
	return []view.Button{{
		Name: "Serve",
	}}
}

func (c *Server) Serve() {
	log.Println("starting http server")
	n := negroni.New()
	// for _, handler := range c.Middleware {
	// 	n.Use(handler)
	// }
	n.UseHandler(c.Handler)
	c.s = &http.Server{
		Handler: n,
	}
	go func() {
		if err := c.s.Serve(c.Listener); err != nil {
			log.Fatal(err)
		}
	}()
}
