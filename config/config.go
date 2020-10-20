// Package config is an interface for dynamic configuration.
package config

import (
	"context"

	"github.com/micro/go-micro/v3/config/loader"
	"github.com/micro/go-micro/v3/config/reader"
	"github.com/micro/go-micro/v3/config/source"
)

// Config is an interface abstraction for dynamic configuration
type Config interface {
	// provide the reader.Values interface
	reader.Values
	// Init the config
	Init(opts ...Option) error
	// Options in the config
	Options() Options
	// Stop the config loader/watcher
	Close() error
	// Load config sources
	Load(source ...source.Source) error
	// Force a source changeset sync
	Sync() error
	// Watch a value for changes
	Watch(path ...string) (Watcher, error)
}

// Watcher is the config watcher
type Watcher interface {
	Next() (reader.Value, error)
	Stop() error
}

type Options struct {
	Loader loader.Loader
	Reader reader.Reader
	Source []source.Source

	// for alternative data
	Context context.Context
}

type Option func(o *Options)

// NewConfig returns new config
func NewConfig(opts ...Option) (Config, error) {
	return newConfig(opts...)
}
