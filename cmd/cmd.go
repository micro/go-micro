// Package cmd is an interface for building a command line binary
package cmd

import (
	"context"

	"github.com/micro/cli/v2"
)

// TODO: replace App with RegisterCommand/RegisterFlags
type Cmd interface {
	// Init initialises options
	// Note: Use Run to parse command line
	Init(opts ...Option) error
	// Options set within this command
	Options() Options
	// The cli app within this cmd
	App() *cli.App
	// Run executes the command
	Run() error
	// Implementation
	String() string
}

type Option func(o *Options)

type Options struct {
	// Name of the application
	Name string
	// Description of the application
	Description string
	// Version of the application
	Version string
	// Action to execute when Run is called and there is no subcommand
	// TODO replace with a build in context
	Action func(*cli.Context) error
	// TODO replace with built in command definition
	Commands []*cli.Command
	// TODO replace with built in flags definition
	Flags []cli.Flag
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Command line Name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Command line Description
func Description(d string) Option {
	return func(o *Options) {
		o.Description = d
	}
}

// Command line Version
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

// Commands to add
func Commands(c ...*cli.Command) Option {
	return func(o *Options) {
		o.Commands = c
	}
}

// Flags to add
func Flags(f ...cli.Flag) Option {
	return func(o *Options) {
		o.Flags = f
	}
}

// Action to execute
func Action(a func(*cli.Context) error) Option {
	return func(o *Options) {
		o.Action = a
	}
}
