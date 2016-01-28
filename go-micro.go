/*
Go micro provides a pluggable library to build microservices.

	import (
		micro "github.com/micro/go-micro"
	)

	service := micro.NewService()
	h := service.Server().NewHandler(&Greeter{})
	service.Server().Handle(h)
	service.Run()


	req := service.Client().NewRequest(service, method, request)
	rsp := response{}
	err := service.Client().Call(req, rsp)

*/

package micro

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"

	"golang.org/x/net/context"
)

type serviceKey struct{}

// Service is an interface that wraps the lower level libraries
// within go-micro. Its a convenience method for building
// and initialising services.
type Service interface {
	Init(...Option)
	Options() Options
	Client() client.Client
	Server() server.Server
	Run() error
	String() string
}

type Option func(*Options)

var (
	HeaderPrefix = "X-Micro-"
)

func NewService(opts ...Option) Service {
	return newService(opts...)
}

func FromContext(ctx context.Context) (Service, bool) {
	s, ok := ctx.Value(serviceKey{}).(Service)
	return s, ok
}

func NewContext(ctx context.Context, s Service) context.Context {
	return context.WithValue(ctx, serviceKey{}, s)
}
