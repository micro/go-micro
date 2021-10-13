package main

import (
	"go-micro.dev/cmd/gomu/cmd"

	// register commands
	_ "go-micro.dev/cmd/gomu/cmd/cli"
)

func main() {
	cmd.Run()
}
