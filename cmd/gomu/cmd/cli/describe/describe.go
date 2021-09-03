package describe

import (
	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/urfave/cli/v2"
)

var flags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:  "format",
		Value: "json",
		Usage: "output a formatted description, e.g. json or yaml",
	},
}

// NewCommand returns a new describe cli command.
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "describe",
		Usage: "Describe a resource",
		Subcommands: []*cli.Command{
			{
				Name:    "service",
				Aliases: []string{"s"},
				Usage:   "Describe a service resource, e.g. " + cmd.App().Name + " describe service helloworld",
				Action:  Service,
				Flags:   flags,
			},
		},
	}
}
