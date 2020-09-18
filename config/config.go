// Package config is an interface for dynamic configuration.
package config

import (
	"time"

	"context"

	goclient "github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/store"
)

type Values interface {
	Get(path string, options ...Option) Value
	Set(path string, val interface{}, options ...Option)
	Delete(path string, options ...Option)
}

// Config is an interface abstraction for dynamic configuration
type Config interface {
	Values
	// Init the config
	Init(opts ...Option) error
}

// NewConfig returns new config
func NewConfig(store store.Store, opts ...Option) (Config, error) {
	return newConfig(store, opts...)
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
	// SecretKey is used to decode secret values when Getting or Setting them.
	SecretKey string

	// Key is used for namespacing purposes:
	// some Config implementations use the Store interface underneath
	// and will use this value to separate between different configs.
	// Ignore if unsure.
	Key string

	// Client and Context are used only for certain implementations,
	// Ignore these if you are unsure.
	Client  goclient.Client
	Context context.Context
}

// Option sets values in Options
type Option func(o *Options)

func Secret(isSecret bool) Option {
	return func(o *Options) {
		o.Secret = isSecret
	}
}

type config struct {
	options *Options
	store   store.Store
}

func newConfig(store store.Store, opts ...Option) (*config, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return &config{
		options: o,
		store:   store,
	}, nil
}

func (c *config) Init(opts ...Option) error {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	c.options = o
	return nil
}

func mergeOptions(old Options, nu ...Option) Options {
	n := Options{
		Secret:    old.Secret,
		SecretKey: old.SecretKey,
		Client:    old.Client,
		Context:   old.Context,
	}
	for _, opt := range nu {
		opt(&n)
	}
	return n
}

func (c *config) Get(path string, options ...Option) Value {
	key := mergeOptions(*c.options, options...).Key

	rec, err := c.store.Read(key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values, _ := NewJSONValues(dat)
	return values.Get(path)
}

func (c *config) Set(path string, val interface{}, options ...Option) {
	key := mergeOptions(*c.options, options...).Key

	rec, err := c.store.Read(key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values, _ := NewJSONValues(dat)
	values.Set(path, val)
	c.store.Write(&store.Record{
		Key:   key,
		Value: values.Bytes(),
	})
}

func (c *config) Delete(path string, options ...Option) {
	key := mergeOptions(*c.options, options...).Key

	rec, err := c.store.Read(key)
	dat := []byte("{}")
	if err != nil || len(rec) == 0 {
		return
	}
	values, _ := NewJSONValues(dat)
	values.Delete(path)
}
