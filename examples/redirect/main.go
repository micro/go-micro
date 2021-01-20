package main

import (
	"log"

	"context"
	"github.com/asim/go-micro/v3"
	api "github.com/asim/go-micro/v3/api/proto"
)

type Redirect struct{}

func (r *Redirect) Url(ctx context.Context, req *api.Request, rsp *api.Response) error {
	rsp.StatusCode = int32(301)
	rsp.Header = map[string]*api.Pair{
		"Location": &api.Pair{
			Key:    "Location",
			Values: []string{"https://google.com"},
		},
	}
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.api.redirect"),
	)

	// parse command line flags
	service.Init()

	service.Server().Handle(
		service.Server().NewHandler(new(Redirect)),
	)

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
