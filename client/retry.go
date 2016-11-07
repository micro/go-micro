package client

type RetryFunc func(err error) bool

// always retry on error
func alwaysRetry(err error) bool {
	return true
}
