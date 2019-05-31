package memory

import (
	"context"

	"github.com/micro/go-config/source"
)

type changeSetKey struct{}

func withData(d []byte, f string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, changeSetKey{}, &source.ChangeSet{
			Data:   d,
			Format: f,
		})
	}
}

// WithChangeSet allows a changeset to be set
func WithChangeSet(cs *source.ChangeSet) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, changeSetKey{}, cs)
	}
}

// WithJson allows the source data to be set to json
func WithJson(d []byte) source.Option {
	return withData(d, "json")
}

// WithYaml allows the source data to be set to yaml
func WithYaml(d []byte) source.Option {
	return withData(d, "yaml")
}
