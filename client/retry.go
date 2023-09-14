package client

import (
	"context"

	"go-micro.dev/v4/errors"
)

// note that returning either false or a non-nil error will result in the call not being retried.
type RetryFunc func(ctx context.Context, req Request, retryCount int, err error) (bool, error)

// RetryAlways always retry on error.
func RetryAlways(ctx context.Context, req Request, retryCount int, err error) (bool, error) {
	return true, nil
}

// RetryOnError retries a request on a 500 or timeout error.
func RetryOnError(ctx context.Context, req Request, retryCount int, err error) (bool, error) {
	if err == nil {
		return false, nil
	}

	e := errors.Parse(err.Error())
	if e == nil {
		return false, nil
	}

	switch e.Code {
	// Retry on timeout, not on 500 internal server error, as that is a business
	// logic error that should be handled by the user.
	case 408:
		return true, nil
	default:
		return false, nil
	}
}
