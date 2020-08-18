package nats

// Options which are used to configure the nats stream
type Options struct {
	ClusterID string
	ClientID  string
	Address   string
}

// Option is a function which configures options
type Option func(o *Options)

// ClusterID sets the cluster id for the nats connection
func ClusterID(id string) Option {
	return func(o *Options) {
		o.ClusterID = id
	}
}

// ClientID sets the client id for the nats connection
func ClientID(id string) Option {
	return func(o *Options) {
		o.ClientID = id
	}
}

// Address of the nats cluster
func Address(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}
