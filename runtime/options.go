package runtime

type CreateOption func(o *CreateOptions)

type CreateOptions struct {
	// command to execute including args
	Command []string
	// Environment to configure
	Env []string
}

// Command specifies the command to execute
func WithCommand(c string, args ...string) CreateOption {
	return func(o *CreateOptions) {
		// set command
		o.Command = []string{c}
		// set args
		o.Command = append(o.Command, args...)
	}
}

// Env sets the created service env
func WithEnv(env []string) CreateOption {
	return func(o *CreateOptions) {
		o.Env = env
	}
}
