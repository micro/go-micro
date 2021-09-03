package cmd

import (
	"fmt"
	"os"

	mcmd "github.com/asim/go-micro/v3/cmd"
	"github.com/urfave/cli/v2"
)

var (
	// DefaultCmd is the default, unmodified root command.
	DefaultCmd Cmd = NewCmd()

	name        string = "gomu"
	description string = "The Go Micro CLI tool"
	version     string = "latest"
)

// Cmd is the interface that wraps the cli app.
//
// Cmd embeds the Cmd interface from the github.com/asim/go-micro/v3/cmd
// package and adds a Run method.
//
// Run runs the cli app within this command and exits on error.
type Cmd interface {
	mcmd.Cmd
	Run() error
}

type cmd struct {
	app  *cli.App
	opts mcmd.Options
}

// App returns the cli app within this command.
func (c *cmd) App() *cli.App {
	return c.app
}

// Options returns the options set within this command.
func (c *cmd) Options() mcmd.Options {
	return c.opts
}

// Init adds options, parses flags and exits on error.
func (c *cmd) Init(opts ...mcmd.Option) error {
	return mcmd.Init(opts...)
}

// Run runs the cli app within this command and exits on error.
func (c *cmd) Run() error {
	return c.app.Run(os.Args)
}

// DefaultOptions returns the options passed to the default command.
func DefaultOptions() mcmd.Options {
	return DefaultCmd.Options()
}

// App returns the cli app within the default command.
func App() *cli.App {
	return DefaultCmd.App()
}

// Register appends commands to the default app.
func Register(cmds ...*cli.Command) {
	app := DefaultCmd.App()
	app.Commands = append(app.Commands, cmds...)
}

// Run runs the cli app within the default command. On error, it prints the
// error message and exits.
func Run() {
	if err := DefaultCmd.Run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// NewCmd returns a new command.
func NewCmd(opts ...mcmd.Option) Cmd {
	options := mcmd.DefaultOptions()

	// Clear the name, version and description parameters from the default
	// options so the options passed may override them.
	options.Name = ""
	options.Version = ""
	options.Description = ""

	for _, o := range opts {
		o(&options)
	}

	if len(options.Name) == 0 {
		options.Name = name
	}
	if len(options.Description) == 0 {
		options.Description = description
	}
	if len(options.Version) == 0 {
		options.Version = version
	}

	c := new(cmd)
	c.opts = options
	c.app = cli.NewApp()
	c.app.Name = c.opts.Name
	c.app.Usage = c.opts.Description
	c.app.Version = c.opts.Version
	c.app.Flags = mcmd.DefaultFlags

	if len(options.Version) == 0 {
		c.app.HideVersion = true
	}

	return c
}
