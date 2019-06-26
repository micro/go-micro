// Package backoff provides backoff functionality
package backoff

import (
	"math"
	"time"
)

func Do(attempts int) time.Duration {
	if attempts == 0 {
		return time.Duration(0)
	}
	return time.Duration(math.Pow(10, float64(attempts))) * time.Millisecond
}
