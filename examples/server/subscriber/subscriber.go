package subscriber

import (
	log "github.com/golang/glog"
	example "github.com/piemapping/go-micro/examples/server/proto/example"
	"golang.org/x/net/context"
)

type Example struct{}

func (e *Example) Handle(ctx context.Context, msg *example.Message) error {
	log.Info("Handler Received message: ", msg.Say)
	return nil
}

func Handler(msg *example.Message) {
	log.Info("Function Received message: ", msg.Say)
}
