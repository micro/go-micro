// Package resolver resolves network ids to addresses
package resolver

// Resolver is network resolver. It's used to find network nodes
// via id to connect to. This is done based on Network.Id().
// Before we can be part of any network, we have to connect to it.
type Resolver interface {
	// Resolve returns a list of addresses for an id
	Resolve(id string) ([]*Record, error)
}

// A resolved record
type Record struct {
	Address string `json:"address"`
}
