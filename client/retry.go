package client

import "golang.org/x/net/context"

type RetryFunc func(ctx context.Context, req Request, retryCount int, err error) (bool, error)

// always retry on error
func alwaysRetry(ctx context.Context, req Request, retryCount int, err error) (bool, error) {
	return true, err
}
