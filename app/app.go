// Package app encapsulates the client, server and other interfaces to provide a complete dapp
package app

// App is an interface for distributed apps
type App interface {
	// Set the current application name
	Name(string)
	// Call an application by name and endpoint
	Call(name, ep string, req, rsp interface{}) error
	// Register a handler e.g a public Go struct/method with signature func(*Request, *Response) error
	Handle(v interface{}) error
	// Run the application
	Run() error
}
