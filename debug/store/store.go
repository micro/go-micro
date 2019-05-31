// Package store stores the various pieces of data
package store

import (
	"errors"
)

type Store interface {
	Read(...ReadOption) ([]*Record, error)
	Write([]*Record) error
	String() string
}

type Record struct {
	Id    string
	Value interface{}
}

type ReadOptions struct {
	Id string
}

type ReadOption func(o *ReadOptions)

var (
	ErrNotFound = errors.New("not found")
)
