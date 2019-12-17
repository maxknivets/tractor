package daemon

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/manifold/tractor/pkg/misc/daemon"
	"github.com/manifold/tractor/pkg/misc/logging/std"
	"github.com/manifold/tractor/pkg/workspace/rpc"
	"github.com/manifold/tractor/pkg/workspace/state"

	_ "github.com/manifold/tractor/com/file"
	_ "github.com/manifold/tractor/com/http"
	_ "github.com/manifold/tractor/com/net"
	_ "github.com/manifold/tractor/com/time"
)

var (
	addr  = flag.String("addr", "localhost:4243", "server listener address")
	proto = flag.String("proto", "websocket", "server listener protocol")
)

func Run() {
	flag.Parse()
	logger := std.NewLogger("", os.Stdout)
	dm := daemon.New([]daemon.Service{
		&state.Service{
			Log: logger,
		},
		&rpc.Service{
			Protocol:   *proto,
			ListenAddr: *addr,
			Log:        logger,
		},
	}...)
	fatal(dm.Run(context.Background()))
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
