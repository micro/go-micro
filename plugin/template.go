package plugin

var (
	tmpl = `
package main

import (
	"github.com/micro/go-micro/plugin"

	"{{.Path}}"
)

var Plugin = plugin.Config{
	Name: "{{.Name}}",
	Type: "{{.Type}}",
	Path: "{{.Path}}",
	NewFunc: {{.Name}}.{{.NewFunc}},
}
`
)
