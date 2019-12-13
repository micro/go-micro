package runtime

import (
	"io"
)

type Option func(o *Options)

// Options configure runtime
type Options struct {
	// Notifier for updates
	Notifier Notifier
	// Service type to manage
	Type string
}

// WithNotifier specifies a notifier for updates
func WithNotifier(n Notifier) Option {
	return func(o *Options) {
		o.Notifier = n
	}
}

// WithType sets the service type to manage
func WithType(t string) Option {
	return func(o *Options) {
		o.Type = t
	}
}

type CreateOption func(o *CreateOptions)

type ReadOption func(o *ReadOptions)

// CreateOptions configure runtime services
type CreateOptions struct {
	// command to execute including args
	Command []string
	// Environment to configure
	Env []string
	// Log output
	Output io.Writer
	// Type of service to create
	Type string
}

// ReadOptions queries runtime services
type ReadOptions struct {
	// Service name
	Service string
	// Version queries services with given version
	Version string
	// Type of service
	Type string
}

// WithCommand specifies the command to execute
func WithCommand(args ...string) CreateOption {
	return func(o *CreateOptions) {
		// set command
		o.Command = args
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

// ReadService returns services with the given name
func ReadService(service string) ReadOption {
	return func(o *ReadOptions) {
		o.Service = service
	}
}

// WithVersion confifgures service version
func ReadVersion(version string) ReadOption {
	return func(o *ReadOptions) {
		o.Version = version
	}
}

// ReadType returns services of the given type
func ReadType(t string) ReadOption {
	return func(o *ReadOptions) {
		o.Type = t
	}
}
