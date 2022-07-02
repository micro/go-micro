package api

func NewOptions(opts ...Option) Options {
	options := Options{
		Address: ":8080",
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}
