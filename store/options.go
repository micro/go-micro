package store

import (
	"context"
)

type Options struct {
	// nodes to connect to
	Nodes []string
	// Namespace of the store
	Namespace string
	// Prefix of the keys used
	Prefix string
	// Suffix of the keys used
	Suffix string
	// Alternative options
	Context context.Context
}

type Option func(o *Options)

// Nodes is a list of nodes used to back the store
func Nodes(a ...string) Option {
	return func(o *Options) {
		o.Nodes = a
	}
}

// Prefix sets a prefix to any key ids used
func Prefix(p string) Option {
	return func(o *Options) {
		o.Prefix = p
	}
}

// Namespace offers a way to have multiple isolated
// stores in the same backend, if supported.
func Namespace(ns string) Option {
	return func(o *Options) {
		o.Namespace = ns
	}
}

// ReadPrefix uses the key as a prefix
func ReadPrefix() ReadOption {
	return func(o *ReadOptions) {
		o.Prefix = true
	}
}

// ReadSuffix uses the key as a suffix
func ReadSuffix() ReadOption {
	return func(o *ReadOptions) {
		o.Suffix = true
	}
}
