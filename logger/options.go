package logger

import (
	"context"
)

// Option for load profiles maybe
// eg. yml
// micro:
//   logger:
//     name:
//     dialect: zap/default/logrus
//     zap:
//       xxx:
//     logrus:
//       xxx:
type Option func(*Options)

type Options struct {
	// The Log Level
	Level Level
	// Other opts
	Context context.Context
}

// WithLevel sets the log level
func WithLevel(l Level) Option {
	return func(o *Options) {
		o.Level = l
	}
}
