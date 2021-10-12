// Package registry uses the go-micro registry for selection
package registry

import (
	"go-micro.dev/v4/cmd"
	"go-micro.dev/v4/selector"
)

func init() {
	cmd.DefaultSelectors["registry"] = NewSelector
}

// NewSelector returns a new registry selector
func NewSelector(opts ...selector.Option) selector.Selector {
	return selector.NewSelector(opts...)
}
