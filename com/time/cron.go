package time

import "github.com/manifold/tractor/pkg/manifold"

func init() {
	manifold.RegisterComponent(&CronManager{}, "")
}

type CronManager struct {
	Hello  string
	Hello2 string
}
