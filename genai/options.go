package genai

// Option sets options for a GenAI provider.
func WithAPIKey(key string) Option {
	return func(o *Options) {
		o.APIKey = key
	}
}

func WithEndpoint(endpoint string) Option {
	return func(o *Options) {
		o.Endpoint = endpoint
	}
}

func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}