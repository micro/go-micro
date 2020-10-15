// Package config is an interface for dynamic configuration.
package config

import (
	"time"
)

// Config is an interface abstraction for dynamic configuration
type Config interface {
	Get(path string, options ...Option) (Value, error)
	Set(path string, val interface{}, options ...Option) error
	Delete(path string, options ...Option) error
}

// Value represents a value of any type
type Value interface {
	Exists() bool
	Bool(def bool) bool
	Int(def int) int
	String(def string) string
	Float64(def float64) float64
	Duration(def time.Duration) time.Duration
	StringSlice(def []string) []string
	StringMap(def map[string]string) map[string]string
	Scan(val interface{}) error
	Bytes() []byte
}

type Options struct {
	Secret bool
}

type Option func(o *Options)

func Secret(b bool) Option {
	return func(o *Options) {
		o.Secret = b
	}
}

type Secrets interface {
	Config
}
