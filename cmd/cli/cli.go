// Package cli is a urfave/cli implementation of the command
package cli

import (
	"os"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v3/cmd"
)

type cliCmd struct {
	opts cmd.Options
	app  *cli.App
}

func (c *cliCmd) Init(opts ...cmd.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}
	c.app.Name = c.opts.Name
	c.app.Description = c.opts.Description
	c.app.Version = c.opts.Version
	c.app.Flags = c.opts.Flags
	c.app.Commands = c.opts.Commands
	c.app.Action = c.opts.Action
	return nil
}

func (c *cliCmd) Options() cmd.Options {
	return c.opts
}

func (c *cliCmd) App() *cli.App {
	return c.app
}

func (c *cliCmd) Run() error {
	return c.app.Run(os.Args)
}

func (c *cliCmd) String() string {
	return "cli"
}

func NewCmd(opts ...cmd.Option) cmd.Cmd {
	c := new(cliCmd)
	c.app = cli.NewApp()
	c.Init(opts...)
	return c
}
