// Package nitro is for blazingly fast distributed app development
package service

import (
	"github.com/asim/nitro/v3/client"
	"github.com/asim/nitro/v3/server"
)

// App is an interface for the Nitro App
type App interface {
	// Init initialises options
	Init(...Option)
	// Options returns the current options
	Options() Options
	// The service name
	Name() string
	// Client is used to call services
	Client() client.Client
	// Server is for handling requests and events
	Server() server.Server
	// Run the service
	Run() error
	// The service implementation
	String() string
}
