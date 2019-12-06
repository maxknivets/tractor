package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/manifold/qtalk/libmux/mux"
	"github.com/manifold/qtalk/qrpc"
	"github.com/manifold/tractor/pkg/agent"
	"github.com/manifold/tractor/pkg/data/icons"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

var (
	tractorUserPath string
	devMode         bool
)

// `tractor agent` command
func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Starts the agent systray app",
		Long:  "Starts the agent systray app.",
		Run:   runAgent,
	}
	cmd.PersistentFlags().BoolVarP(&devMode, "dev", "d", false, "run in debug mode")
	cmd.PersistentFlags().StringVarP(&tractorUserPath, "path", "p", "", "path to the user tractor directory (default is ~/.tractor)")
	cmd.AddCommand(agentCallCmd())
	return cmd
}

func runAgent(cmd *cobra.Command, args []string) {
	ag := openAgent()

	if agentSockExists(ag) && devMode {
		return
	}

	go func(a *agent.Agent) {
		fatal(agent.ListenAndServe(a))
	}(ag)

	systray.Run(onReady(ag), ag.Shutdown)
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

func onReady(ag *agent.Agent) func() {
	return func() { buildSystray(ag) }
}

func buildSystray(ag *agent.Agent) {
	systray.SetIcon(icons.Tractor)
	systray.SetTooltip("Tractor")

	workspaces, err := ag.Workspaces()
	fatal(err)

	for _, ws := range workspaces {
		openItem := systray.AddMenuItem(ws.Name, "Open workspace")

		ws.OnStatusChange(func(ws *agent.Workspace) {
			openItem.SetIcon(ws.Status.Icon())
		})

		go func(mi *systray.MenuItem, ws *agent.Workspace) {
			for {
				<-openItem.ClickedCh
				open.StartWith(ws.TargetPath, "Visual Studio Code.app")
			}
		}(openItem, ws)
	}

	systray.AddSeparator()
	mQuitOrig := systray.AddMenuItem("Shutdown", "Quit and shutdown all workspaces")
	go func(mi *systray.MenuItem) {
		<-mi.ClickedCh
		systray.Quit()
	}(mQuitOrig)

	<-sigQuit.Done()
	systray.Quit()
}

// `tractor agent call` command
func agentCallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Makes a QRPC call to the agent app",
		Long:  "Makes a QRPC call to the agent app.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "connect",
		Short: "Connects to a running workspace",
		Long:  "Connects to a workspace, starting it if it is not running. The output is streamed to STDOUT.",
		Args:  cobra.ExactArgs(1),
		Run:   runAgentCall("connect"),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Restarts a workspace",
		Long:  "Starts a workspace, restarting it if it is currently running. The output is streamed to STDOUT.",
		Args:  cobra.ExactArgs(1),
		Run:   runAgentCall("start"),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stops a workspace",
		Long:  "Stops a workspace.",
		Args:  cobra.ExactArgs(1),
		Run:   runAgentCall("stop"),
	})
	return cmd
}

func runAgentCall(callmethod string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		wspath := args[0]
		start := time.Now()
		_, err := agentQRPCCall(os.Stdout, callmethod, wspath)
		if err != nil && err != io.EOF {
			fmt.Printf("qrpc: %s [%s(%q) %s]\n", err, callmethod, wspath, time.Since(start))
			os.Exit(1)
			return
		}
		fmt.Printf("qrpc: %s(%q) %s\n", callmethod, wspath, time.Since(start))
	}
}

func agentQRPCCall(w io.Writer, cmd, wspath string) (string, error) {
	ag := openAgent()
	sess, err := mux.DialUnix(ag.SocketPath)
	if err != nil {
		return "", err
	}

	client := &qrpc.Client{Session: sess}
	var msg string
	resp, err := client.Call(cmd, wspath, &msg)
	if err != nil {
		return msg, err
	}

	if len(msg) > 0 {
		fmt.Fprintf(w, "REPLY => %s\n", msg)
	}

	if resp.Hijacked {
		go func() {
			<-sigQuit.Done()
			resp.Channel.Close()
		}()

		_, err = io.Copy(w, resp.Channel)
		resp.Channel.Close()
		if err != nil && err != io.EOF {
			fmt.Fprintln(w, err)
		}
		fmt.Fprintln(w)
	}

	return msg, nil
}
