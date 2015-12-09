package registry

// Selector builds on the registry as a mechanism to pick nodes
// and mark their status. This allows host pools and other things
// to be built using various algorithms.
type Selector interface {
	Select(service string, opts ...SelectOption) (SelectNext, error)
	Mark(service string, node *Node, err error)
	Reset(service string)
	Close() error
}

// SelectNext is a function that returns the next node
// based on the selector's algorithm
type SelectNext func() (*Node, error)

type SelectorOptions struct {
	Registry Registry
}

type SelectOptions struct {
	Filters []func([]*Service) []*Service
}

// Option used to initialise the selector
type SelectorOption func(*SelectorOptions)

// Option used when making a select call
type SelectOption func(*SelectOptions)

// SelectorRegistry sets the registry used by the selector
func SelectorRegistry(r Registry) SelectorOption {
	return func(o *SelectorOptions) {
		o.Registry = r
	}
}

// SelectFilter adds a filter function to the list of filters
// used during the Select call.
func SelectFilter(fn func([]*Service) []*Service) SelectOption {
	return func(o *SelectOptions) {
		o.Filters = append(o.Filters, fn)
	}
}
