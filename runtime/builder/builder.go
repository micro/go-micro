package builder

import "io"

// Builder is an interface for building packages
type Builder interface {
	// Build a package
	Build(src io.Reader, opts ...Option) (io.Reader, error)
}
