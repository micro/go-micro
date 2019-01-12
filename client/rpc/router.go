package rpc

import (
	"context"

	"github.com/micro/go-micro/client"
)

type router struct{}

func (r *router) SendRequest(context.Context, client.Request) (client.Response, error) {
	return nil, nil
}

func NewRouter() *router {
	return &router{}
}
