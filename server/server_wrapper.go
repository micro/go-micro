package server

import (
	"golang.org/x/net/context"
)

type HandlerFunc func(ctx context.Context, req interface{}, rsp interface{}) error

type SubscriberFunc func(ctx context.Context, msg interface{}) error

type HandlerWrapper func(HandlerFunc) HandlerFunc

type SubscriberWrapper func(SubscriberFunc) SubscriberFunc
