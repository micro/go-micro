package router

// AddPolicy defines routing table addition policy
type AddPolicy int

const (
	// Override overrides existing routing table entry
	OverrideIfExists AddPolicy = iota
	// ErrIfExists returns error if the entry already exists
	ErrIfExists
)

// EntryOptions defines micro network routing table entry options
type EntryOptions struct {
	// DestAddr is destination address
	DestAddr string
	// Hop is the next route hop
	Hop Router
	// SrcAddr defines local routing address
	// On local networkss, this will be the address of local router
	SrcAddr string
	// Metric is route cost metric
	Metric int
	// Policy defines entry addition policy
	Policy AddPolicy
}

// DestAddr sets destination address
func DestAddr(a string) EntryOption {
	return func(o *EntryOptions) {
		o.DestAddr = a
	}
}

// Hop allows to set the route entry options
func Hop(r Router) EntryOption {
	return func(o *EntryOptions) {
		o.Hop = r
	}
}

// SrcAddr sets source address
func SrcAddr(a string) EntryOption {
	return func(o *EntryOptions) {
		o.SrcAddr = a
	}
}

// Metric sets entry metric
func Metric(m int) EntryOption {
	return func(o *EntryOptions) {
		o.Metric = m
	}
}

// AddEntryPolicy sets add entry policy
func AddEntryPolicy(p AddPolicy) EntryOption {
	return func(o *EntryOptions) {
		o.Policy = p
	}
}

// Entry is routing table entry
type Entry interface {
	// Options returns entry options
	Options() EntryOptions
}

type entry struct {
	opts EntryOptions
}

// NewEntry returns new routing table entry
func NewEntry(opts ...EntryOption) Entry {
	eopts := EntryOptions{}

	for _, o := range opts {
		o(&eopts)
	}

	return &entry{
		opts: eopts,
	}
}

// Options returns entry options
func (e *entry) Options() EntryOptions {
	return e.opts
}
