package runtime

import (
	"io"
)

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

// WithEnv sets the created service env
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
