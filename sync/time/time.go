// Package time provides clock synchronization
package time

import (
	"context"
	"time"
)

// Time returns synchronized time
type Time interface {
	Now() (time.Time, error)
}

type Options struct {
	Context context.Context
}

type Option func(o *Options)
