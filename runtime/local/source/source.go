// Package source retrieves source code
package source

// Source retrieves source code
type Source interface {
	// Fetch repo from a url
	Fetch(url string) (*Repository, error)
	// Commit and upload repo
	Commit(*Repository) error
	// The sourcerer
	String() string
}

// Repository is the source repository
type Repository struct {
	// Name or repo
	Name string
	// Local path where repo is stored
	Path string
	// URL from which repo was retrieved
	URL string
}
