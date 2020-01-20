package stdlib

import (
	"net"

	"github.com/manifold/tractor/stdlib/file"
	"github.com/manifold/tractor/stdlib/http"
	"github.com/manifold/tractor/stdlib/net/irc"
	"github.com/manifold/tractor/stdlib/time"

	"github.com/manifold/tractor/pkg/manifold/library"
)

func Load() {
	// file
	library.Register(&file.Local{}, "", "")
	library.Register(&file.Path{}, "", "")

	// http
	library.Register(&http.SingleUserBasicAuth{}, "", "")
	library.Register(&http.FileServer{}, "", "")
	library.Register(&http.Logger{}, "", "")
	library.Register(&http.Mux{}, "", "")
	library.Register(&http.Server{}, "", "")
	library.Register(&http.TemplateRenderer{}, "", "")

	// net
	library.Register(&net.TCPListener{}, "", "")

	// net/irc
	library.Register(&irc.IRCClient{}, "", "")
	library.Register(&irc.BangMux{}, "", "")

	// time
	library.Register(&time.CronManager{}, "", "")

}
