package plugin

var (
	tmpl = `
package main

import (
	"{{.Path}}"
	"github.com/micro/go-micro/v3/plugins"
)

var Plugin = plugin.Plugin{
	Name: "{{.Name}}",
	Type: "{{.Type}}",
	Path: "{{.Path}}",
	NewFunc: {{.Name}}.{{.NewFunc}},
}
`
)
