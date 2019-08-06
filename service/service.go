// Package service encapsulates the client, server and other interfaces to provide a complete micro service.
package service

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
)

type Service interface {
	Init(...Option)
	Options() Options
	Client() client.Client
	Server() server.Server
	Run() error
	String() string
}
