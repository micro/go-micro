package main

import (
	"github.com/asim/go-micro/cmd/gomu/cmd"

	// register commands
	_ "github.com/asim/go-micro/cmd/gomu/cmd/cli"
)

func main() {
	cmd.Run()
}
