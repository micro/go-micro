package storeconfig

import (
	"github.com/micro/go-micro/v3/config"
	"github.com/micro/go-micro/v3/store"
)

// NewConfig returns new config
func NewConfig(store store.Store, key string) (config.Config, error) {
	return newConfig(store, key)
}

type conf struct {
	key   string
	store store.Store
}

func newConfig(store store.Store, key string) (*conf, error) {
	return &conf{
		store: store,
		key:   key,
	}, nil
}

func mergeOptions(old config.Options, nu ...config.Option) config.Options {
	n := config.Options{
		Secret: old.Secret,
	}
	for _, opt := range nu {
		opt(&n)
	}
	return n
}

func (c *conf) Get(path string, options ...config.Option) (config.Value, error) {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values := config.NewJSONValues(dat)
	return values.Get(path), nil
}

func (c *conf) Set(path string, val interface{}, options ...config.Option) error {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values := config.NewJSONValues(dat)
	values.Set(path, val)
	return c.store.Write(&store.Record{
		Key:   c.key,
		Value: values.Bytes(),
	})
}

func (c *conf) Delete(path string, options ...config.Option) error {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err != nil || len(rec) == 0 {
		return nil
	}
	values := config.NewJSONValues(dat)
	values.Delete(path)
	return c.store.Write(&store.Record{
		Key:   c.key,
		Value: values.Bytes(),
	})
}
