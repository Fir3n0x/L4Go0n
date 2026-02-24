package cmd

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

// Handle updates between front and back, send data to the front

// Global variable
var TempRenderer = &TemplateRenderer{
	templates: template.Must(template.ParseGlob("templates/*.html")),
}

// Renderer structure
type TemplateRenderer struct {
	templates *template.Template
}

// Render updates
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
