package provider

// Option returns a function which sets an option
type Option func(*Options)

// Options a provider can have
type Options struct {
	// ClientID is the application's ID.
	ClientID string
	// ClientSecret is the application's secret.
	ClientSecret string
	// Endpoint for the provider
	Endpoint string
	// Redirect url incase of UI
	Redirect string
	// Scope of the oauth request
	Scope string
}

// Credentials is an option which sets the client id and secret
func Credentials(id, secret string) Option {
	return func(o *Options) {
		o.ClientID = id
		o.ClientSecret = secret
	}
}

// Endpoint sets the endpoint option
func Endpoint(e string) Option {
	return func(o *Options) {
		o.Endpoint = e
	}
}

// Redirect sets the Redirect option
func Redirect(r string) Option {
	return func(o *Options) {
		o.Redirect = r
	}
}

// Scope sets the oauth scope
func Scope(s string) Option {
	return func(o *Options) {
		o.Scope = s
	}
}
