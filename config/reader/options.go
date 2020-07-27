package reader

import (
	"github.com/micro/go-micro/v3/config/encoder"
	"github.com/micro/go-micro/v3/config/encoder/hcl"
	"github.com/micro/go-micro/v3/config/encoder/json"
	"github.com/micro/go-micro/v3/config/encoder/toml"
	"github.com/micro/go-micro/v3/config/encoder/xml"
	"github.com/micro/go-micro/v3/config/encoder/yaml"
)

type Options struct {
	Encoding              map[string]encoder.Encoder
	DisableReplaceEnvVars bool
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Encoding: map[string]encoder.Encoder{
			"json": json.NewEncoder(),
			"yaml": yaml.NewEncoder(),
			"toml": toml.NewEncoder(),
			"xml":  xml.NewEncoder(),
			"hcl":  hcl.NewEncoder(),
			"yml":  yaml.NewEncoder(),
		},
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

func WithEncoder(e encoder.Encoder) Option {
	return func(o *Options) {
		if o.Encoding == nil {
			o.Encoding = make(map[string]encoder.Encoder)
		}
		o.Encoding[e.String()] = e
	}
}

// WithDisableReplaceEnvVars disables the environment variable interpolation preprocessor
func WithDisableReplaceEnvVars() Option {
	return func(o *Options) {
		o.DisableReplaceEnvVars = true
	}
}
