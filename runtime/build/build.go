package build

// Build is runtime service build
type Build struct {
	// Commit is git commit sha
	Commit string `json:"commit,omitempty"`
	// Image is Docker build timestamp
	Image string `json:"image"`
	// Release is micro release tag
	Release string `json:"release,omitempty"`
}
