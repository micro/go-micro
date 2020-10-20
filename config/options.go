package config

import (
	"github.com/asim/go-micro/v3/config/loader"
	"github.com/asim/go-micro/v3/config/reader"
	"github.com/asim/go-micro/v3/config/source"
)

// WithLoader sets the loader for manager config
func WithLoader(l loader.Loader) Option {
	return func(o *Options) {
		o.Loader = l
	}
}

// WithSource appends a source to list of sources
func WithSource(s source.Source) Option {
	return func(o *Options) {
		o.Source = append(o.Source, s)
	}
}

// WithReader sets the config reader
func WithReader(r reader.Reader) Option {
	return func(o *Options) {
		o.Reader = r
	}
}
