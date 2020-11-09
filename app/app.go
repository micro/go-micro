// Package app encapsulates the client, server and other interfaces to provide a complete dapp
package app

// App is an interface for distributed apps
type App interface {
	// The service name
	Name() string
	// Init initialises options
	Init(...Option)
	// Options returns the current options
	Options() Options
	// Call an application
	Call(name, ep string, req, rsp interface{}) error
	// Register a handler e.g a Go struct
	Handle(v interface{}) error
	// Run the application
	Run() error
	// The service implementation
	String() string
}
