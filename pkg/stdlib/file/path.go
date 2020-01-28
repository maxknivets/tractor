package file

import (
	"net/http"
)

type Path struct {
	Filepath string
}

func (c *Path) Open(name string) (http.File, error) {
	return http.Dir(c.Filepath).Open(name)
}
