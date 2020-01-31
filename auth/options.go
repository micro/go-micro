package auth

type Options struct {
	PublicKey string
}

type Option func(o *Options)

// PublicKey is the JWT public key
func PublicKey(key string) Option {
	return func(o *Options) {
		o.PublicKey = key
	}
}
