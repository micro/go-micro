package client

import (
	"context"
	"math"
	"time"
)

type BackoffFunc func(ctx context.Context, req Request, attempts int) (time.Duration, error)

// exponential backoff multiplied by a factor of 0.1 second.
func exponentialBackoff(ctx context.Context, req Request, attempts int) (time.Duration, error) {
	return time.Duration(math.Pow(float64(attempts), math.E)) * time.Millisecond * 100, nil
}
