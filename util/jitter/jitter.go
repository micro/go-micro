// Package jitter provides a random jitter
package jitter

import (
	"math/rand"
	"sync"
	"time"
)

var (
	r  = rand.New(rand.NewSource(time.Now().UnixNano()))
	mu sync.Mutex
)

// Do returns a random time to jitter with max cap specified
func Do(d time.Duration) time.Duration {
	mu.Lock()
	v := r.Float64()
	mu.Unlock()
	v *= float64(d.Nanoseconds())
	return time.Duration(v)
}
