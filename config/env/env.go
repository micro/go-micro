// Package env provides config from environment variables
package env

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/asim/go-micro/v3/config"
)

type envConfig struct{}

// NewConfig returns new config
func NewConfig() (*envConfig, error) {
	return new(envConfig), nil
}

func formatKey(v string) string {
	if len(v) == 0 {
		return ""
	}

	v = strings.ToUpper(v)
	return strings.Replace(v, ".", "_", -1)
}

func (c *envConfig) Get(path string, options ...config.Option) (config.Value, error) {
	v := os.Getenv(formatKey(path))
	if len(v) == 0 {
		v = "{}"
	}
	return config.NewJSONValue([]byte(v)), nil
}

func (c *envConfig) Set(path string, val interface{}, options ...config.Option) error {
	key := formatKey(path)
	v, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return os.Setenv(key, string(v))
}

func (c *envConfig) Delete(path string, options ...config.Option) error {
	v := formatKey(path)
	return os.Unsetenv(v)
}
