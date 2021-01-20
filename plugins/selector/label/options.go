package label

import (
	"context"

	"github.com/asim/go-micro/v3/selector"
)

type labelKey struct{}

type label struct {
	key string
	val string
}

// Label used in the priority label list
func Label(k, v string) selector.Option {
	return func(o *selector.Options) {
		l, ok := o.Context.Value(labelKey{}).([]label)
		if !ok {
			l = []label{}
		}
		l = append(l, label{k, v})
		o.Context = context.WithValue(o.Context, labelKey{}, l)
	}
}
