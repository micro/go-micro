package stream

import (
	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/urfave/cli/v2"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "stream",
		Usage: "Create a service stream",
		Subcommands: []*cli.Command{
			{
				Name:    "bidi",
				Aliases: []string{"b"},
				Usage:   "Create a bidirectional service stream, e.g. " + cmd.App().Name + " stream bidirectional helloworld Helloworld.PingPong '{\"stroke\": 1}' '{\"stroke\": 2}'",
				Action:  Bidirectional,
			},
			{
				Name:    "server",
				Aliases: []string{"s"},
				Usage:   "Create a server service stream, e.g. " + cmd.App().Name + " stream server helloworld Helloworld.ServerStream '{\"count\": 10}'",
				Action:  Server,
			},
		},
	})
}
