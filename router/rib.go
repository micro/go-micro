package router

// RIB is Routing Information Base
type RIB interface {
	// Initi initializes RIB
	Init(...RIBOption) error
	// Options returns RIB options
	Options() RIBOptions
	// Routes returns routes in RIB
	Routes() []Route
	// String returns debug info
	String() string
}

// RIBOptions allow to set RIB sources.
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
