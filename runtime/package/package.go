// Package packager creates a binary image. Due to package being a reserved keyword we use packager.
package packager

import (
	"github.com/micro/go-micro/runtime/source"
)

// Package builds binaries
type Packager interface {
	// Compile builds a binary
	Compile(*Source) (*Binary, error)
	// Deletes the binary
	Delete(*Binary) error
}

// Source is the source of a build
type Source struct {
	// Language is the language of code
	Language string
	// Location of the source
	Repository *source.Repository
}

// Binary is the representation of a binary
type Binary struct {
	// Name of the binary
	Name string
	// Location of the binary
	Path string
	// Type of binary
	Type string
	// Source of the binary
	Source *Source
}
