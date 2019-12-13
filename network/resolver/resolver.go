// Package resolver resolves network names to addresses
package resolver

// Resolver is network resolver. It's used to find network nodes
// via the name to connect to. This is done based on Network.Name().
// Before we can be part of any network, we have to connect to it.
type Resolver interface {
	// Resolve returns a list of addresses for a name
	Resolve(name string) ([]*Record, error)
}

// A resolved record
type Record struct {
	Address  string `json:"address"`
	Priority int64  `json:"priority"`
}
