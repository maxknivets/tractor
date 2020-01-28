package stdlib

import (
	"path"
	"runtime"

	"github.com/manifold/tractor/pkg/stdlib/file"
	"github.com/manifold/tractor/pkg/stdlib/http"
	"github.com/manifold/tractor/pkg/stdlib/net"
	"github.com/manifold/tractor/pkg/stdlib/net/irc"
	"github.com/manifold/tractor/pkg/stdlib/time"

	"github.com/manifold/tractor/pkg/manifold/library"
)

func filepath(subpath string) string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(filename), subpath)
}

func Load() {
	// file
	library.Register(&file.Local{}, "", filepath("file/local.go"))
	library.Register(&file.Path{}, "", filepath("file/path.go"))
	library.Register(&file.Explorer{}, "", filepath("file/explorer.go"))

	// http
	library.Register(&http.SingleUserBasicAuth{}, "", filepath("http/basicauth.go"))
	library.Register(&http.FileServer{}, "", filepath("http/fileserver.go"))
	library.Register(&http.Logger{}, "", filepath("http/logger.go"))
	library.Register(&http.Mux{}, "", filepath("http/mux.go"))
	library.Register(&http.Server{}, "", filepath("http/server.go"))
	library.Register(&http.TemplateRenderer{}, "", filepath("http/templaterenderer.go"))

	// net
	library.Register(&net.TCPListener{}, "", filepath("net/listener.go"))

	// net/irc
	library.Register(&irc.IRCClient{}, "", filepath("net/irc/irc.go"))
	library.Register(&irc.BangMux{}, "", filepath("net/irc/irc.go"))

	// time
	library.Register(&time.CronManager{}, "", filepath("time/cron.go"))

}
