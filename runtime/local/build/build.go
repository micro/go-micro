// Package build builds a micro runtime package
package build

import (
	"github.com/asim/go-micro/v3/runtime/local/source"
)

// Builder builds binaries
type Builder interface {
	// Build builds a package
	Build(*Source) (*Package, error)
	// Clean deletes the package
	Clean(*Package) error
}

// Source is the source of a build
type Source struct {
	// Language is the language of code
	Language string
	// Location of the source
	Repository *source.Repository
}

// Package is micro service package
type Package struct {
	// Name of the binary
	Name string
	// Location of the binary
	Path string
	// Type of binary
	Type string
	// Source of the binary
	Source *Source
}
