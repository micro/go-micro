// Package jitter provides a random jitter
package jitter

import (
	"math/rand"
	"time"
)

var (
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// Do returns a random time to jitter with max cap specified
func Do(d time.Duration) time.Duration {
	v := r.Float64() * float64(d.Nanoseconds())
	return time.Duration(v)
}
