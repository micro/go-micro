package network

type Option func(*Options)

// Options configure network
type Options struct {
	// Id of the node
	Id string
	// Name of the network
	Name string
	// Address to bind to
	Address string
	// Advertise sets the address to advertise
	Advertise string
	// Nodes is a list of nodes to connect to
	Nodes []string
}

// Id sets the id of the network node
func Id(id string) Option {
	return func(o *Options) {
		o.Id = id
	}
}

// Name sets the network name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Address sets the network address
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// Advertise sets the address to advertise
func Advertise(a string) Option {
	return func(o *Options) {
		o.Advertise = a
	}
}

// Nodes is a list of nodes to connect to
func Nodes(n ...string) Option {
	return func(o *Options) {
		o.Nodes = n
	}
}
