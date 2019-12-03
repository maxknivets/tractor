package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "tractor",
		Short: "Tractor",
		Long:  "Tractor",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// context that cancels when an os signal to quit the app has been received.
	sigQuit context.Context
)

func main() {
	rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(agentCmd())
	rootCmd.AddCommand(devCmd())

	ct, cancelFunc := context.WithCancel(context.Background())
	sigQuit = ct

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGHUP)

	go func(c <-chan os.Signal) {
		<-c
		cancelFunc()
	}(c)
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
