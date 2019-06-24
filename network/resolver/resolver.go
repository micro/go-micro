// Package resolver resolves network ids to addresses
package resolver

type Resolver interface {
	// Resolve returns a list of addresses for an id
	Resolve(id string) ([]*Record, error)
}

// A resolved record
type Record struct {
	Address string `json:"address"`
}
