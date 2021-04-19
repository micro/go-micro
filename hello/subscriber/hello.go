package subscriber

import (
	"context"
	log "github.com/micro/go-micro/v2/logger"

	hello "hello/proto/hello"
)

type Hello struct{}

func (e *Hello) Handle(ctx context.Context, msg *hello.Message) error {
	log.Info("Handler Received message: ", msg.Say)
	return nil
}

func Handler(ctx context.Context, msg *hello.Message) error {
	log.Info("Function Received message: ", msg.Say)
	return nil
}
