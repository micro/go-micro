package logger

import "context"

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
	Context context.Context
}
