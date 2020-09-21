// Package handler provides http handlers
package handler

import (
	"net/http"
)

// Handler represents a HTTP handler that manages a request
type Handler interface {
	// standard http handler
	http.Handler
	// name of handler
	String() string
}
