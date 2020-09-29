package secrets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/micro/go-micro/v3/config"
)

// NewSecrets returns a config that encrypts values at rest
func NewSecrets(config config.Config, encryptionKey string) (config.Secrets, error) {
	return newSecrets(config, encryptionKey)
}

type secretConf struct {
	config        config.Config
	encryptionKey string
}

func newSecrets(config config.Config, encryptionKey string) (*secretConf, error) {
	return &secretConf{
		config:        config,
		encryptionKey: encryptionKey,
	}, nil
}

func (c *secretConf) Get(path string, options ...config.Option) (config.Value, error) {
	val, err := c.config.Get(path, options...)
	empty := config.NewJSONValue([]byte("null"))
	if err != nil {
		return empty, err
	}
	var v interface{}
	err = json.Unmarshal(val.Bytes(), &v)
	if err != nil {
		return empty, err
	}
	v, err = convertElements(v, c.fromEncrypted)
	if err != nil {
		return empty, err
	}
	dat, err := json.Marshal(v)
	if err != nil {
		return empty, err
	}
	return config.NewJSONValue(dat), nil
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
		// This bit decides if the Secrets implementation suppports nonencrypted values
		// ie. we could do:
		// return nil, fmt.Errorf("Encrypted values should be strings, but got: %v", elem)
		// but let's go with not making nonencrypted values blow up the whole thing
		return elem, nil
	}
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return elem, nil
	}
	decrypted, err := decrypt(string(dec), []byte(c.encryptionKey))
	if err != nil {
		return elem, nil
	}
	var ret interface{}
	return ret, json.Unmarshal([]byte(decrypted), &ret)
}
