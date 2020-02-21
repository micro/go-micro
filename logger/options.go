package logger

import (
	"context"
	"io"
)

type Option func(*Options)

type Options struct {
	// The Log Level
	Level Level
	// Other opts
	Context context.Context
}

type fieldsKey struct{}

func WithFields(fields Fields) Option {
	return setOption(fieldsKey{}, fields)
}

type levelKey struct{}

func WithLevel(lvl Level) Option {
	return setOption(levelKey{}, lvl)
}

type outputKey struct{}

func WithOutput(out io.Writer) Option {
	return setOption(outputKey{}, out)
}

func setOption(k, v interface{}) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
