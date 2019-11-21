package main

import (
	_ "github.com/manifold/tractor/dev/workspace/delegates"
	"github.com/manifold/tractor/pkg/workspace/workspaceprocess"
)

func main() {
	workspaceprocess.Run()
}
