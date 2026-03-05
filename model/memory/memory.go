// Package memory provides an in-memory model.Model implementation.
// This is the same as model.NewModel() but importable as a standalone package.
package memory

import (
	"go-micro.dev/v5/model"
)

// New creates a new in-memory model.
func New(opts ...model.Option) model.Model {
	return model.NewModel(opts...)
}
