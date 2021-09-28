package template

// Plugins is the plugins template used for new projects.
var Plugins = `package main

import (
	_ "github.com/asim/go-micro/plugins/registry/kubernetes/v3"
)
`
