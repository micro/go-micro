// Package backoff provides backoff functionality
package backoff

import (
	"math"
	"time"
)

func Do(attempts int) time.Duration {
	if attempts > 13 {
		return 2 * time.Minute
	}
	return time.Duration(math.Pow(float64(attempts), math.E)) * time.Millisecond * 100
}
