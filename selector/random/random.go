package random

import (
	"github.com/micro/go-micro/v2/selector"
)

// NewSelector returns a random selector
func NewSelector(opts ...selector.Option) selector.Selector {
	return selector.DefaultSelector
}
