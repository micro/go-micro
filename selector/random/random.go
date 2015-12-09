package random

import (
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/selector"
)

func init() {
	cmd.Selectors["random"] = NewSelector
}

func NewSelector(opts ...selector.Option) selector.Selector {
	return selector.NewSelector(opts...)
}
