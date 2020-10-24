// Package config is an interface for dynamic configuration.
package config

import (
	"context"

	"github.com/asim/go-micro/v3/config/loader"
	"github.com/asim/go-micro/v3/config/reader"
	"github.com/asim/go-micro/v3/config/source"
)

// Config is an interface abstraction for dynamic configuration
type Config interface {
	// Init the config
	Init(opts ...Option) error
	// Options in the config
	Options() Options
	// Load config sources
	Load(path ...string) (Values, error)
	// Watch a value for changes
	Watch(path ...string) (Watcher, error)
	// Force a source changeset sync
	Sync() error
	// Stop the config loader/watcher
	Close() error
}

// Watcher is the config watcher
type Watcher interface {
	Next() (Value, error)
	Stop() error
}

// Values is returned by the reader
type Values interface {
        Bytes() []byte
        Get(path ...string) Value
        Set(val interface{}, path ...string)
        Del(path ...string)
        Map() map[string]interface{}
        Scan(v interface{}) error
}

// Value represents a value of any type
type Value interface {
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
	Loader loader.Loader
	Reader reader.Reader
	Source []source.Source

	// for alternative data
	Context context.Context
}

type Option func(o *Options)
