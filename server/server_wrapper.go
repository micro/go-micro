package server

import (
	"golang.org/x/net/context"
)

type HandlerFunc func(ctx context.Context, req interface{}, rsp interface{}) error

type Wrapper func(HandlerFunc) HandlerFunc
