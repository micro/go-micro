// Package model provides data access models
package model

import (
	"time"

	"github.com/micro/go-micro/v2/codec"
	"github.com/micro/go-micro/v2/store"
)

type Model interface {
	// Initialise the options
	Init(...Option) error
	// Retrieve the options
	Options() Options
	// String is the type of model e.g cache, document
	String() string
}

// Record is the common record stored by all models
type Record struct {
	// Unique id
	Id string
	// Timestamp
	Timestamp time.Time
	// Serialised Data
	Data []byte
	// Associated metadata
	Metadata map[string]interface{}
}

type Options struct {
	// The codec for encoding/decoding
	Codec codec.Marshaler
	// The store used for underlying storage
	Store store.Store
}

type Option func(o *Options) error
