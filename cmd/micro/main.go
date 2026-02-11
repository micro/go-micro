package main

import (
	"embed"
	"go-micro.dev/v5/cmd"

	_ "go-micro.dev/v5/cmd/micro/cli"
	_ "go-micro.dev/v5/cmd/micro/cli/build"
	_ "go-micro.dev/v5/cmd/micro/cli/deploy"
	_ "go-micro.dev/v5/cmd/micro/mcp"
	_ "go-micro.dev/v5/cmd/micro/run"
	"go-micro.dev/v5/cmd/micro/server"
)

//go:embed web/styles.css web/main.js web/templates/*
var webFS embed.FS

var version = "5.0.0-dev"

func init() {
	server.HTML = webFS
}

func main() {
	cmd.Init(
		cmd.Name("micro"),
		cmd.Version(version),
	)
}
