package describe

import (
	"github.com/urfave/cli/v2"
	mcli "go-micro.dev/v4/cmd/micro/cli"
)

var flags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:  "format",
		Value: "json",
		Usage: "output a formatted description, e.g. json or yaml",
	},
}

func init() {
	mcli.Register(&cli.Command{
		Name:  "describe",
		Usage: "Describe a resource",
		Subcommands: []*cli.Command{
			{
				Name:    "service",
				Aliases: []string{"s"},
				Usage:   "Describe a service resource, e.g. " + mcli.App().Name + " describe service helloworld",
				Action:  Service,
				Flags:   flags,
			},
		},
	})
}
