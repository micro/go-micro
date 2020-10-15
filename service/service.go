// Package service encapsulates the client, server and other interfaces to provide a complete micro service.
package service

import (
	"github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/model"
	"github.com/micro/go-micro/v3/server"
)

// Service is an interface for a micro service
type Service interface {
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
	// Model is used to access data
	Model() model.Model
	// Run the service
	Run() error
	// The service implementation
	String() string
}
