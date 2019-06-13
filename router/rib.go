package router

// RIB is Routing Information Base.
// RIB is used to source the base routing table.
type RIB interface {
	// Initi initializes RIB
	Init(...RIBOption) error
	// Options returns RIB options
	Options() RIBOptions
	// Routes returns routes
	Routes() []Route
	// String returns debug info
	String() string
}

// RIBOptopn sets RIB options
type RIBOption func(*RIBOptions)

// RIBOptions configures various RIB options
type RIBOptions struct {
	// Source defines RIB source URL
	Source string
}

// Source sets RIB source
func Source(s string) RIBOption {
	return func(o *RIBOptions) {
		o.Source = s
	}
}
