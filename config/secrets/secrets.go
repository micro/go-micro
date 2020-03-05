// Package secrets is for loading secrets from config
package secrets

import (
	"github.com/micro/go-micro/config/encoder"
	"github.com/micro/go-micro/config/reader"
	"github.com/micro/go-micro/config/source"
)

type Secrets interface {
	// load a secret
	Load(path ...string) (reader.Values, error)
}

type Options struct {
	// Source to load from
	Source source.Source

	// used to decode secrets in config
	Encoder encoder.Encoder
}
