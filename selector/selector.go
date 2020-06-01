package selector

import (
	"errors"

	"github.com/micro/go-micro/v2/router"
)

var (
	// DefaultSelector is the default selector
	DefaultSelector = newSelector()

	// ErrNoneAvailable is returned by select when no routes were provided to select from
	ErrNoneAvailable = errors.New("none available")
)

// Selector selects a route from a pool
type Selector interface {
	// Init a selector with options
	Init(...Option) error
	// Options the selector is using
	Options() Options
	// Select a route from the pool using the strategy
	Select(...router.Route) (*router.Route, error)
	// Record the error returned from a route to inform future selection
	Record(*router.Route, error) error
	// String returns the name of the selector
	String() string
}

// Options used to configure a selector
type Options struct{}

// Option updates the options
type Option func(*Options)
