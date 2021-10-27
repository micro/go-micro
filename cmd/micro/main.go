package main

import (
	"github.com/asim/go-micro/cmd/micro/cmd"

	// register commands
	_ "github.com/asim/go-micro/cmd/micro/cmd/cli"
)

func main() {
	cmd.Run()
}
