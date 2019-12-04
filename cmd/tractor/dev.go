package main

import (
	"os"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"
)

// `tractor selfdev`
func selfdevCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "selfdev",
		Short: "Starts the dev harness for Tractor",
		Long:  "Starts the dev harness for Tractor",
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				cmd := exec.Command("dev/bin/tractor", "agent", "--dev")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cmd := exec.Command("tsc", "-watch", "--preserveWatchOutput", "-p", "./extension")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
				wg.Done()
			}()
			wg.Wait()
		},
	}
}
