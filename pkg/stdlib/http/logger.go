package http

import (
	"net/http"

	"github.com/urfave/negroni"
)

type Logger struct {
	logger *negroni.Logger
}

func (c *Logger) Initialize() error {
	c.logger = negroni.NewLogger()
	return nil
}

func (c *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	c.logger.ServeHTTP(w, r, next)
}
