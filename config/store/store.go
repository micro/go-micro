package storeconfig

import (
	"github.com/micro/go-micro/v3/config"
	"github.com/micro/go-micro/v3/store"
)

// NewConfig returns new config
func NewConfig(store store.Store, key string) (config.Config, error) {
	return newConfig(store)
}

type conf struct {
	key   string
	store store.Store
}

func newConfig(store store.Store) (*conf, error) {
	return &conf{
		store: store,
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

func (c *conf) Get(path string, options ...config.Option) config.Value {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values := config.NewJSONValues(dat)
	return values.Get(path)
}

func (c *conf) Set(path string, val interface{}, options ...config.Option) {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	values := config.NewJSONValues(dat)
	values.Set(path, val)
	c.store.Write(&store.Record{
		Key:   c.key,
		Value: values.Bytes(),
	})
}

func (c *conf) Delete(path string, options ...config.Option) {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	if err != nil || len(rec) == 0 {
		return
	}
	values := config.NewJSONValues(dat)
	values.Delete(path)
}
