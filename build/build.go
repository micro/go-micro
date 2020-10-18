// Package build is for building source into a package
package build

// Build is an interface for building packages
type Build interface {
	// Package builds a package
	Package(*Source) (*Package, error)
	// Remove removes the package
	Remove(*Package) error
	// Implementation of build
	String() string
}

// Source is the source of a build
type Source struct {
	// Name of the source
	Name string
	// Path to the source if local
	Path string
	// Language is the language of code
	Language string
	// Location of the source
	Repository string
}

// Package is packaged format for source
type Package struct {
	// Name of the package
	Name string
	// Location of the package
	Path string
	// Type of package e.g tarball, binary, docker
	Type string
	// Source of the package
	Source *Source
}
