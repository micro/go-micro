package server

import (
	"golang.org/x/net/context"
)

type HandlerFunc func(ctx context.Context, req Request, rsp interface{}) error

type SubscriberFunc func(ctx context.Context, msg Publication) error

type HandlerWrapper func(HandlerFunc) HandlerFunc

type SubscriberWrapper func(SubscriberFunc) SubscriberFunc
