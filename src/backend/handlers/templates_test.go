package handlers

import (
	"html/template"
	"testing"
)

func TestFrontendTemplatesParse(t *testing.T) {
	templates := []string{
		"../../frontend/index.html",
		"../../frontend/history.html",
	}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			if _, err := template.ParseFiles(tmpl); err != nil {
				t.Fatalf("template.ParseFiles(%q) returned error: %v", tmpl, err)
			}
		})
	}
}
