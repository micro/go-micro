package s3

// Options used to configure the s3 blob store
type Options struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Secure          bool
}

// Option configures one or more options
type Option func(o *Options)

// Endpoint sets the endpoint option
func Endpoint(e string) Option {
	return func(o *Options) {
		o.Endpoint = e
	}
}

// Region sets the region option
func Region(r string) Option {
	return func(o *Options) {
		o.Region = r
	}
}

// Credentials sets the AccessKeyID and SecretAccessKey options
func Credentials(id, secret string) Option {
	return func(o *Options) {
		o.AccessKeyID = id
		o.SecretAccessKey = secret
	}
}

// Insecure sets the secure option to false. It is enabled by default.
func Insecure() Option {
	return func(o *Options) {
		o.Secure = false
	}
}
