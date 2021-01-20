package memory

import (
	"github.com/asim/go-micro/v3/config/loader"
	"github.com/asim/go-micro/v3/config/reader"
	"github.com/asim/go-micro/v3/config/source"
)

// WithSource appends a source to list of sources
func WithSource(s source.Source) loader.Option {
	return func(o *loader.Options) {
		o.Source = append(o.Source, s)
	}
}

// WithReader sets the config reader
func WithReader(r reader.Reader) loader.Option {
	return func(o *loader.Options) {
		o.Reader = r
	}
}
