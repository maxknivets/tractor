package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/rjeczalik/notify"
	"github.com/spf13/cobra"
)

// `tractor selfdev`
func selfdevCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "selfdev",
		Short: "Starts the dev harness for Tractor",
		Long:  "Starts the dev harness for Tractor",
		Run: func(cmd *cobra.Command, args []string) {
			make := exec.Command("make", "build")
			make.Stdout = os.Stdout
			make.Stderr = os.Stderr
			make.Run()
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

func notifyChanges(dir string, exts []string, onlyCreate bool, cb func(path string)) {
	c := make(chan notify.EventInfo, 1)
	types := notify.All
	if onlyCreate {
		types = notify.Create
	}
	if err := notify.Watch(dir, c, types); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)
	for event := range c {
		path := event.Path()
		dir, file := filepath.Split(path)
		if filepath.Base(dir) == ".git" {
			continue
		}
		if filepath.Base(file)[0] == '.' {
			continue
		}
		if extensionIn(path, exts) {
			cb(path)
		}
	}
}

func extensionIn(path string, exts []string) bool {
	for _, ext := range exts {
		if filepath.Ext(path) == ext {
			return true
		}
	}
	return false
}
