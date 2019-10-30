package runtime

import (
	"io"

	"github.com/micro/go-micro/runtime/poller"
)

// Options define runtime options
type Option func(o *Options)

// Option is runtime option
type Options struct {
	// Type defines type of runtime
	Type string
	// Poller polls updates
	Poller poller.Poller
}

// Type defines type of runtime
func Type(t string) Option {
	return func(o *Options) {
		o.Type = t
	}
}

// AutoUpdate enables micro auto-updates
func AutoUpdate(p poller.Poller) Option {
	return func(o *Options) {
		o.Poller = p
	}
}

type CreateOption func(o *CreateOptions)

type CreateOptions struct {
	// command to execute including args
	Command []string
	// Environment to configure
	Env []string
	// Log output
	Output io.Writer
}

// WithCommand specifies the command to execute
func WithCommand(c string, args ...string) CreateOption {
	return func(o *CreateOptions) {
		// set command
		o.Command = []string{c}
		// set args
		o.Command = append(o.Command, args...)
	}
}

// WithEnv sets the created service environment
func WithEnv(env []string) CreateOption {
	return func(o *CreateOptions) {
		o.Env = env
	}
}

// WithOutput sets the arg output
func WithOutput(out io.Writer) CreateOption {
	return func(o *CreateOptions) {
		o.Output = out
	}
}
