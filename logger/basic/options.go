package basic

import (
	"context"
	"io"

	"github.com/micro/go-micro/v2/logger"
)

type Options struct {
	logger.Options
}

type fieldsKey struct{}

func WithFields(fields logger.Fields) logger.Option {
	return setOption(fieldsKey{}, fields)
}

type levelKey struct{}

func WithLevel(lvl logger.Level) logger.Option {
	return setOption(levelKey{}, lvl)
}

type outputKey struct{}

func WithOutput(out io.Writer) logger.Option {
	return setOption(outputKey{}, out)
}

func setOption(k, v interface{}) logger.Option {
	return func(o *logger.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
