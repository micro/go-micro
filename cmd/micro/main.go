package main

import (
	"embed"
	"go-micro.dev/v6/cmd"

	_ "go-micro.dev/v6/cmd/micro/a2a"
	_ "go-micro.dev/v6/cmd/micro/api"
	_ "go-micro.dev/v6/cmd/micro/chat"
	_ "go-micro.dev/v6/cmd/micro/cli"
	_ "go-micro.dev/v6/cmd/micro/cli/build"
	_ "go-micro.dev/v6/cmd/micro/cli/deploy"
	_ "go-micro.dev/v6/cmd/micro/flow"
	_ "go-micro.dev/v6/cmd/micro/mcp"
	_ "go-micro.dev/v6/cmd/micro/resource"
	_ "go-micro.dev/v6/cmd/micro/run"
	"go-micro.dev/v6/cmd/micro/server"
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
