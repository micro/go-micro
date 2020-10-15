package store

import (
	"encoding/json"
	"strings"

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

	// marshal to JSON and back so we can iterate on the
	// value without reflection
	// @todo only do this if a struct
	JSON, err := json.Marshal(val)
	if err != nil {
		return err
	}
	var v interface{}
	err = json.Unmarshal(JSON, &v)
	if err != nil {
		return err
	}

	m, ok := v.(map[string]interface{})
	if ok {
		err := traverse(m, []string{path}, func(p string, value interface{}) error {
			values.Set(p, value)
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		values.Set(path, val)
	}
	return c.store.Write(&store.Record{
		Key:   c.key,
		Value: values.Bytes(),
	})
}

func traverse(m map[string]interface{}, paths []string, callback func(path string, value interface{}) error) error {
	for k, v := range m {
		val, ok := v.(map[string]interface{})
		if !ok {
			err := callback(strings.Join(append(paths, k), "."), v)
			if err != nil {
				return err
			}
			continue
		}
		err := traverse(val, append(paths, k), callback)
		if err != nil {
			return err
		}
	}
	return nil
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
