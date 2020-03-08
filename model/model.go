// Package model provides data access models
package model

import (
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

type Options struct {
	// The codec for encoding/decoding
	Codec codec.Marshaler
	// The store used for underlying storage
	Store store.Store
}

type Option func(o *Options) error
