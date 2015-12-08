package client

import (
	"github.com/micro/go-micro/registry"
)

// Selector takes a Registry and returns a NodeSelector.
// Used by the client to initialise a selector.
type Selector func(registry.Registry) NodeSelector
