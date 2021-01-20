package client

import (
	"context"
	"time"

	"github.com/asim/go-micro/v3/util/backoff"
)

type BackoffFunc func(ctx context.Context, req Request, attempts int) (time.Duration, error)

func exponentialBackoff(ctx context.Context, req Request, attempts int) (time.Duration, error) {
	return backoff.Do(attempts), nil
}
