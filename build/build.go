// Package build is for building source into a package
package build

// Builder is an interface for building packages
type Builder interface {
	// Package builds a package
	Build(src Source, opts ...Option) (Build, error)
}

// Package is packaged source
type Package interface {
	// Location of the package, e.g micro/foo-api:latest
	Location() string
}

// Source code to be built
type Source interface {
	// String to describe the source, e.g. foo/api
	String() string
}

// Status defines the status of a build
type Status int

const (
	// Unknown is returned by the builder when the status is unknown
	Unknown Status = iota
	// Pending indicates the builder hasn't starting building the source yet
	Pending
	// Building indicates the build is in progress
	Building
	// Uploading is returned if the builder is uploading the source to a remote repository. Not
	// all builders will return this status for a build.
	Uploading
	// Failed indicates there was an error building the source. See the Error for more information.
	Failed
	// Completed indicates the build completed okay
	Completed
)

// Build of source
type Build interface {
	// Package contains the result of the build. It will return nil whilst the build status is not
	// complete.
	Package() Package
	// Status returns the status of the build
	Status() Status
	// Error returns the build error if present
	Error() error
	// Wait will wait for the build to either complete or fail
	Wait()
}
