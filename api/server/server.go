// Package server provides an API gateway server which handles inbound requests
package server

import (
	"net/http"
)

// Server serves api requests
type Server interface {
	Address() string
	Init(opts ...Option) error
	Handle(path string, handler http.Handler)
	Start() error
	Stop() error
}
