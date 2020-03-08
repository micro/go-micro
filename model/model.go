// Package model provides data access models
package model

type Model interface {
	// Initialise the options
	Init(...Option) error
	// Retrieve the options
	Options() Options
	// String is the type of model e.g cache, document
	String() string
}

type Options struct{}

type Option func(o *Options) error
