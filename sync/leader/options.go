package leader

type Options struct {
	Nodes []string
	Group string
}

type ElectOptions struct{}

// Nodes sets the addresses of the underlying systems
func Nodes(a ...string) Option {
	return func(o *Options) {
		o.Nodes = a
	}
}

// Group sets the group name for coordinating leadership
func Group(g string) Option {
	return func(o *Options) {
		o.Group = g
	}
}
