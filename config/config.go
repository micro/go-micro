// Package config is an interface for dynamic configuration.
package config

import (
	"time"

	"context"

	goclient "github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/store"
)

type Values interface {
	Get(path string) Value
	Set(val interface{}, path string)
	Delete(path string)
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
	Secret  bool
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

func (c *config) Get(path string) Value {
	key := "micro"
	// @todo support tables
	rec, err := c.store.Read(key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values, _ := NewJSONValues(dat)
	return values.Get(path)
}

func (c *config) Set(val interface{}, path string) {
	key := "micro"
	// @todo support tables
	rec, err := c.store.Read(key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values, _ := NewJSONValues(dat)
	values.Set(val, path)
	c.store.Write(&store.Record{
		Key:   key,
		Value: values.Bytes(),
	})
}

func (c *config) Delete(path string) {
	// @todo support tables
	key := "micro"
	rec, err := c.store.Read(key)
	dat := []byte("{}")
	if err != nil || len(rec) == 0 {
		return
	}
	values, _ := NewJSONValues(dat)
	values.Delete(path)
}
