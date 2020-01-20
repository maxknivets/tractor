package http

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type TemplateRenderer struct {
	Template fmt.Stringer
}

func (c *TemplateRenderer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.Template == nil {
		return
	}
	t, err := template.New("").Parse(c.Template.String())
	if err != nil {
		log.Println(err)
	}
	err = t.Execute(w, map[string]interface{}{
		"Request": r,
	})
	if err != nil {
		log.Println(err)
	}
}
