package http

import (
	"net/http"

	"github.com/goji/httpauth"
)

type SingleUserBasicAuth struct {
	Username string
	Password string
}

func (c *SingleUserBasicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	httpauth.SimpleBasicAuth(c.Username, c.Password)(next).ServeHTTP(w, r)
}
