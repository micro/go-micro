package auth

type Options struct {
	// Token is an auth token
	Token string
	// Public key base64 encoded
	PublicKey string
	// Private key base64 encoded
	PrivateKey string
	// Endpoints to exclude
	Exclude []string
}

type Option func(o *Options)

// Exclude ecludes a set of endpoints from authorization
func Exclude(e ...string) Option {
	return func(o *Options) {
		o.Exclude = e
	}
}

// PublicKey is the JWT public key
func PublicKey(key string) Option {
	return func(o *Options) {
		o.PublicKey = key
	}
}

// PrivateKey is the JWT private key
func PrivateKey(key string) Option {
	return func(o *Options) {
		o.PrivateKey = key
	}
}

// Token sets an auth token
func Token(t string) Option {
	return func(o *Options) {
		o.Token = t
	}
}

type GenerateOptions struct {
	// Metadata associated with the account
	Metadata map[string]string
	// Roles/scopes associated with the account
	Roles []*Role
}

type GenerateOption func(o *GenerateOptions)

// Metadata for the generated account
func Metadata(md map[string]string) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Metadata = md
	}
}

// Roles for the generated account
func Roles(rs []*Role) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Roles = rs
	}
}

// NewGenerateOptions from a slice of options
func NewGenerateOptions(opts ...GenerateOption) GenerateOptions {
	var options GenerateOptions
	for _, o := range opts {
		o(&options)
	}

	return options
}
