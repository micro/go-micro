package store

import (
	"context"
	"time"
)

// Options contains configuration for the Store
type Options struct {
	// Nodes contains the addresses or other connection information of the backing storage.
	// For example, an etcd implementation would contain the nodes of the cluster.
	// A SQL implementation could contain one or more connection strings.
	Nodes []string
	// Namespace allows multiple isolated stores to be kept in one backend, if supported.
	// For example multiple tables in a SQL store.
	Namespace string
	// Prefix sets a global prefix on all keys
	Prefix string
	// Suffix sets a global suffix on all keys
	Suffix string
	// Context should contain all implementation specific options, using context.WithValue.
	Context context.Context
}

// Option sets values in Options
type Option func(o *Options)

// Nodes contains the addresses or other connection information of the backing storage.
// For example, an etcd implementation would contain the nodes of the cluster.
// A SQL implementation could contain one or more connection strings.
func Nodes(a ...string) Option {
	return func(o *Options) {
		o.Nodes = a
	}
}

// Namespace allows multiple isolated stores to be kept in one backend, if supported.
// For example multiple tables in a SQL store.
func Namespace(ns string) Option {
	return func(o *Options) {
		o.Namespace = ns
	}
}

// Prefix sets a global prefix on all keys
func Prefix(p string) Option {
	return func(o *Options) {
		o.Prefix = p
	}
}

// Suffix sets a global suffix on all keys
func Suffix(s string) Option {
	return func(o *Options) {
		o.Suffix = s
	}
}

// WithContext sets the stores context, for any extra configuration
func WithContext(c context.Context) Option {
	return func(o *Options) {
		o.Context = c
	}
}

// ReadOptions configures an individual Read operation
type ReadOptions struct {
	// Prefix returns all records that are prefixed with key
	Prefix bool
	// Suffix returns all records that have the suffix key
	Suffix bool
	// Limit limits the number of returned records
	Limit uint
	// Offset when combined with Limit supports pagination
	Offset uint
}

// ReadOption sets values in ReadOptions
type ReadOption func(r *ReadOptions)

// ReadPrefix returns all records that are prefixed with key
func ReadPrefix() ReadOption {
	return func(r *ReadOptions) {
		r.Prefix = true
	}
}

// ReadSuffix returns all records that have the suffix key
func ReadSuffix() ReadOption {
	return func(r *ReadOptions) {
		r.Suffix = true
	}
}

// ReadLimit limits the number of responses to l
func ReadLimit(l uint) ReadOption {
	return func(r *ReadOptions) {
		r.Limit = l
	}
}

// ReadOffset starts returning responses from o. Use in conjunction with Limit for pagination
func ReadOffset(o uint) ReadOption {
	return func(r *ReadOptions) {
		r.Offset = o
	}
}

// WriteOptions configures an individual Write operation
// If Expiry and TTL are set TTL takes precedence
type WriteOptions struct {
	// Expiry is the time the record expires
	Expiry time.Time
	// TTL is the time until the record expires
	TTL time.Duration
}

// WriteOption sets values in WriteOptions
type WriteOption func(w *WriteOptions)

// WriteExpiry is the time the record expires
func WriteExpiry(t time.Time) WriteOption {
	return func(w *WriteOptions) {
		w.Expiry = t
	}
}

// WriteTTL is the time the record expires
func WriteTTL(d time.Duration) WriteOption {
	return func(w *WriteOptions) {
		w.TTL = d
	}
}

// DeleteOptions configures an individual Delete operation
type DeleteOptions struct{}

// DeleteOption sets values in DeleteOptions
type DeleteOption func(d *DeleteOptions)

// ListOptions configures an individual List operation
type ListOptions struct {
	// Prefix returns all keys that are prefixed with key
	Prefix string
	// Suffix returns all keys that end with key
	Suffix string
	// Limit limits the number of returned keys
	Limit uint
	// Offset when combined with Limit supports pagination
	Offset uint
}

// ListOption sets values in ListOptions
type ListOption func(l *ListOptions)

// ListPrefix returns all keys that are prefixed with key
func ListPrefix(p string) ListOption {
	return func(l *ListOptions) {
		l.Prefix = p
	}
}

// ListSuffix returns all keys that end with key
func ListSuffix(s string) ListOption {
	return func(l *ListOptions) {
		l.Suffix = s
	}
}

// ListLimit limits the number of returned keys to l
func ListLimit(l uint) ListOption {
	return func(lo *ListOptions) {
		lo.Limit = l
	}
}

// ListOffset starts returning responses from o. Use in conjunction with Limit for pagination.
func ListOffset(o uint) ListOption {
	return func(l *ListOptions) {
		l.Offset = o
	}
}
