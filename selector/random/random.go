package random

import (
	"github.com/micro/go-micro/selector"
)

func NewSelector(opts ...selector.Option) selector.Selector {
	return selector.NewSelector(opts...)
}
