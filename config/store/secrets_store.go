package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/micro/go-micro/v3/config"
	"github.com/micro/go-micro/v3/store"
)

// NewSecrets returns a config that encrypts values at rest
func NewSecrets(store store.Store, key, encryptionKey string) (config.Config, error) {
	return newSecrets(store, key, encryptionKey)
}

type secretConf struct {
	store         store.Store
	config        config.Config
	encryptionKey string
	key           string
}

func newSecrets(store store.Store, key, encryptionKey string) (*secretConf, error) {
	c, err := NewConfig(store, key)
	if err != nil {
		return nil, err
	}
	return &secretConf{
		store:         store,
		config:        c,
		key:           key,
		encryptionKey: encryptionKey,
	}, nil
}

func (c *secretConf) Get(path string, options ...config.Option) (config.Value, error) {
	rec, err := c.store.Read(c.key)
	dat := []byte("{}")
	empty := config.NewJSONValue([]byte("null"))

	if err == nil && len(rec) > 0 {
		dat = rec[0].Value
	}
	var v interface{}
	err = json.Unmarshal(dat, &v)
	if err != nil {
		return empty, err
	}
	v, err = convertElements(v, c.fromEncrypted)
	if err != nil {
		return empty, err
	}
	dat, err = json.Marshal(v)
	if err != nil {
		return empty, err
	}
	values := config.NewJSONValues(dat)
	return values.Get(path), nil
}

func (c *secretConf) Set(path string, val interface{}, options ...config.Option) error {
	// marshal to JSON and back so we can iterate on the
	// value without reflection
	JSON, err := json.Marshal(val)
	if err != nil {
		return err
	}
	var v interface{}
	err = json.Unmarshal(JSON, &v)
	if err != nil {
		return err
	}
	v, err = convertElements(v, c.toEncrypted)
	if err != nil {
		return err
	}
	return c.config.Set(path, v)
}

func (c *secretConf) Delete(path string, options ...config.Option) error {
	return c.config.Delete(path, options...)
}

func convertElements(elem interface{}, conversionFunc func(elem interface{}) (interface{}, error)) (interface{}, error) {
	switch m := elem.(type) {
	case map[string]interface{}:
		for k, v := range m {
			conv, err := convertElements(v, conversionFunc)
			if err != nil {
				return nil, err
			}
			m[k] = conv

		}
		return m, nil
	}

	return conversionFunc(elem)
}

func (c *secretConf) toEncrypted(elem interface{}) (interface{}, error) {
	dat, err := json.Marshal(elem)
	if err != nil {
		return nil, err
	}
	encrypted, err := encrypt(string(dat), []byte(c.encryptionKey))
	if err != nil {
		return nil, fmt.Errorf("Failed to encrypt: %v", err)
	}
	return string(base64.StdEncoding.EncodeToString([]byte(encrypted))), nil
}

func (c *secretConf) fromEncrypted(elem interface{}) (interface{}, error) {
	s, ok := elem.(string)
	if !ok {
		return nil, fmt.Errorf("Encrypted values should be strings, but got: %v", elem)
	}
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("Badly encoded secret")
	}
	decrypted, err := decrypt(string(dec), []byte(c.encryptionKey))
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt: %v", err)
	}
	var ret interface{}
	return ret, json.Unmarshal([]byte(decrypted), &ret)
}
