package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/manifold/qtalk/libmux/mux"
	"github.com/manifold/qtalk/qrpc"
	"github.com/manifold/tractor/pkg/agent"
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
	}
	cmd.PersistentFlags().BoolVarP(&devMode, "dev", "d", false, "run in debug mode")
	cmd.PersistentFlags().StringVarP(&tractorUserPath, "path", "p", "", "path to the user tractor directory (default is ~/.tractor)")
	cmd.AddCommand(agentCallCmd())
	return cmd
}

// `tractor agent call` command
func agentCallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Makes a QRPC call to the agent app",
		Long:  "Makes a QRPC call to the agent app.",
		Args:  cobra.ExactArgs(2),
		Run:   runAgentCall(),
	}

	// cmd.AddCommand(&cobra.Command{
	// 	Use:   "connect",
	// 	Short: "Connects to a running workspace",
	// 	Long:  "Connects to a workspace, starting it if it is not running. The output is streamed to STDOUT.",
	// 	Args:  cobra.ExactArgs(1),
	// 	Run:   runAgentCall("connect"),
	// })
	// cmd.AddCommand(&cobra.Command{
	// 	Use:   "start",
	// 	Short: "Restarts a workspace",
	// 	Long:  "Starts a workspace, restarting it if it is currently running. The output is streamed to STDOUT.",
	// 	Args:  cobra.ExactArgs(1),
	// 	Run:   runAgentCall("start"),
	// })
	// cmd.AddCommand(&cobra.Command{
	// 	Use:   "stop",
	// 	Short: "Stops a workspace",
	// 	Long:  "Stops a workspace.",
	// 	Args:  cobra.ExactArgs(1),
	// 	Run:   runAgentCall("stop"),
	// })
	return cmd
}

func runAgentCall() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		callmethod := args[0]
		arg := args[1]
		start := time.Now()
		_, err := agentQRPCCall(os.Stdout, callmethod, arg)
		if err != nil && err != io.EOF {
			fmt.Printf("qrpc: %s [%s(%q) %s]\n", err, callmethod, arg, time.Since(start))
			os.Exit(1)
			return
		}
		fmt.Printf("qrpc: %s(%q) %s\n", callmethod, arg, time.Since(start))
	}
}

func agentQRPCCall(w io.Writer, cmd, arg string) (string, error) {
	var sess mux.Session
	var err error
	if os.Getenv("QRPC_HOST") != "" {
		sess, err = mux.DialWebsocket(os.Getenv("QRPC_HOST"))
	} else {
		ag := openAgent()
		sess, err = mux.DialUnix(ag.SocketPath)
	}
	if err != nil {
		return "", err
	}

	client := &qrpc.Client{Session: sess}
	var msg string
	resp, err := client.Call(cmd, arg, &msg)
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

func openAgent() *agent.Agent {
	ag, err := agent.Open(tractorUserPath, nil, false)
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
