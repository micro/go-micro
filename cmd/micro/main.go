package main

import (
	"go-micro.dev/v4/cmd/micro/cli"

	// register commands
	_ "go-micro.dev/v4/cmd/micro/cli/call"
	_ "go-micro.dev/v4/cmd/micro/cli/describe"
	_ "go-micro.dev/v4/cmd/micro/cli/generate"
	_ "go-micro.dev/v4/cmd/micro/cli/new"
	_ "go-micro.dev/v4/cmd/micro/cli/run"
	_ "go-micro.dev/v4/cmd/micro/cli/services"
	_ "go-micro.dev/v4/cmd/micro/cli/stream"
)

func main() {
	cli.Run()
}
