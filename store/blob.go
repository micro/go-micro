package store

import (
	"errors"
	"io"
)

var (
	// ErrMissingKey is returned when no key is passed to blob store Read / Write
	ErrMissingKey = errors.New("missing key")
)

// BlobStore is an interface for reading / writing blobs
type BlobStore interface {
	Read(key string, opts ...BlobOption) (io.Reader, error)
	Write(key string, blob io.Reader, opts ...BlobOption) error
	Delete(key string, opts ...BlobOption) error
}

// BlobOptions contains options to use when interacting with the store
type BlobOptions struct {
	// Namespace to  from
	Namespace string
}

// BlobOption sets one or more BlobOptions
type BlobOption func(o *BlobOptions)

// BlobNamespace sets the Namespace option
func BlobNamespace(ns string) BlobOption {
	return func(o *BlobOptions) {
		o.Namespace = ns
	}
}
