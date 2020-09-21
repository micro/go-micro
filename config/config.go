// Package config is an interface for dynamic configuration.
package config

import (
	"time"

	"github.com/micro/go-micro/v3/store"
)

// Config is an interface abstraction for dynamic configuration
type Config interface {
	Get(path string, options ...Option) Value
	Set(path string, val interface{}, options ...Option)
	Delete(path string, options ...Option)
}

// NewConfig returns new config
func NewConfig(store store.Store, key string) (Config, error) {
	return newConfig(store)
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
	// Is the value being read a secret?
	// If true, the Config will try to decode it with `SecretKey`
	Secret bool
}

// Option sets values in Options
type Option func(o *Options)

func Secret(isSecret bool) Option {
	return func(o *Options) {
		o.Secret = isSecret
	}
}

type config struct {
	key   string
	store store.Store
}

func newConfig(store store.Store) (*config, error) {
	return &config{
		store: store,
	}, nil
}

func mergeOptions(old Options, nu ...Option) Options {
	n := Options{
		Secret: old.Secret,
	}
	for _, opt := range nu {
		opt(&n)
	}
	return n
}

func (c *config) Get(path string, options ...Option) Value {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values, _ := NewJSONValues(dat)
	return values.Get(path)
}

func (c *config) Set(path string, val interface{}, options ...Option) {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values, _ := NewJSONValues(dat)
	values.Set(path, val)
	c.store.Write(&store.Record{
		Key:   c.key,
		Value: values.Bytes(),
	})
}

func (c *config) Delete(path string, options ...Option) {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err != nil || len(rec) == 0 {
		return
	}
	values, _ := NewJSONValues(dat)
	values.Delete(path)
}
