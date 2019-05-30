package memory

import (
	"context"

	"github.com/micro/go-micro/config/source"
)

type changeSetKey struct{}

// WithChangeSet allows a changeset to be set
func WithChangeSet(cs *source.ChangeSet) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, changeSetKey{}, cs)
	}
}

// WithData allows the source data to be set
func WithData(d []byte) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, changeSetKey{}, &source.ChangeSet{
			Data:   d,
			Format: "json",
		})
	}
}
