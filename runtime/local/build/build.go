// Package build builds a micro runtime package
package build

import (
	"go-micro.dev/v4/runtime/local/source"
)

// Builder builds binaries.
type Builder interface {
	// Build builds a package
	Build(*Source) (*Package, error)
	// Clean deletes the package
	Clean(*Package) error
}

// Source is the source of a build.
type Source struct {
	// Location of the source
	Repository *source.Repository
	// Language is the language of code
	Language string
}

// Package is micro service package.
type Package struct {
	// Source of the binary
	Source *Source
	// Name of the binary
	Name string
	// Location of the binary
	Path string
	// Type of binary
	Type string
}
