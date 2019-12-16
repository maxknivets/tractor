package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/agent/console"
	"github.com/manifold/tractor/pkg/agent/rpc"
	"github.com/manifold/tractor/pkg/agent/selfdev"
	"github.com/manifold/tractor/pkg/agent/systray"
	"github.com/manifold/tractor/pkg/agent/systray/subprocess"
	"github.com/manifold/tractor/pkg/misc/daemon"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "tractor-agent",
		Short: "Tractor Agent",
		Long:  "Tractor Agent",
		Run:   runAgent,
	}

	tractorUserPath string
	devMode         bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&devMode, "dev", "d", false, "run in debug mode")
	rootCmd.PersistentFlags().StringVarP(&tractorUserPath, "path", "p", "", "path to the user tractor directory (default is ~/.tractor)")
}

func main() {
	rootCmd.Execute()
}

func runAgent(cmd *cobra.Command, args []string) {
	if os.Getenv("SYSTRAY_SUBPROCESS") != "" {
		subprocess.Run()
		return
	}

	ctx := context.Background()
	ag := openAgent()

	if agentSockExists(ag) && devMode {
		fmt.Println("Agent will not run in dev mode if agent socket exists.")
		return
	}

	dm := daemon.New(services(ag)...)
	fatal(dm.Run(ctx))
}

func services(a *agent.Agent) []daemon.Service {
	services := []daemon.Service{
		a,
		a.Console,
		&systray.Service{},
		&rpc.Service{},
	}
	if devMode {
		services = append(services, []daemon.Service{
			&selfdev.Service{},
		}...)
	}
	return services
}

func openAgent() *agent.Agent {
	ag, err := agent.Open(tractorUserPath, console.New(), devMode)
	fatal(err)
	return ag
}

func agentSockExists(ag *agent.Agent) bool {
	_, err := os.Stat(ag.SocketPath)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
