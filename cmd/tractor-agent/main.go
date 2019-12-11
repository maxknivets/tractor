package main

import (
	"context"
	"log"
	"os"

	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/agent/logger"
	"github.com/manifold/tractor/pkg/agent/rpc"
	"github.com/manifold/tractor/pkg/agent/selfdev"
	"github.com/manifold/tractor/pkg/agent/systray"
	"github.com/manifold/tractor/pkg/agent/systray/subprocess"
	"github.com/manifold/tractor/pkg/daemon"
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

	ag := openAgent()
	if agentSockExists(ag) && devMode {
		log.Println("Agent will not run in dev mode if agent socket exists.")
		return
	}
	services := []daemon.Service{
		logger.New(),
		&systray.Service{Agent: ag},
		&rpc.Service{Agent: ag},
	}
	if devMode {
		services = append(services, []daemon.Service{
			&selfdev.Service{Agent: ag},
		}...)
	}
	dm := daemon.New(services...)
	fatal(dm.Run(context.Background()))
}

func openAgent() *agent.Agent {
	ag, err := agent.Open(tractorUserPath)
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
