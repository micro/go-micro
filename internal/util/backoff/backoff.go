// Package backoff provides backoff functionality
package backoff

import (
	"math"
	"time"
)

// Do is a function x^e multiplied by a factor of 0.1 second.
// Result is limited to 2 minute.
func Do(attempts int) time.Duration {
	if attempts > 13 {
		return 2 * time.Minute
	}
	return time.Duration(math.Pow(float64(attempts), math.E)) * time.Millisecond * 100
}
