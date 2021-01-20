// Package registry uses the go-micro registry for selection
package registry

import (
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/selector"
)

func init() {
	cmd.DefaultSelectors["registry"] = NewSelector
}

// NewSelector returns a new registry selector
func NewSelector(opts ...selector.Option) selector.Selector {
	return selector.NewSelector(opts...)
}
