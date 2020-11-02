// Package app encapsulates the client, server and other interfaces to provide a complete dapp
package app

import (
	"github.com/asim/nitro/v3/client"
	"github.com/asim/nitro/v3/server"
)

// App is an interface for distributed apps
type App interface {
	// The service name
	Name() string
	// Init initialises options
	Init(...Option)
	// Options returns the current options
	Options() Options
	// Client is used to call services
	Client() client.Client
	// Server is for handling requests and events
	Server() server.Server
	// Run the service
	Run() error
	// The service implementation
	String() string
}
