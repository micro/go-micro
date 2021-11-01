package main

import (
	"go-micro.dev/v4/cmd/micro/cmd"

	// register commands
	_ "go-micro.dev/v4/cmd/micro/cmd/cli"
)

func main() {
	cmd.Run()
}
