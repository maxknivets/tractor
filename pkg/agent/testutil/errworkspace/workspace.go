package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "boomtown", os.Getpid())
	os.Exit(1)
}
