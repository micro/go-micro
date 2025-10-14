package template

var (
	HandlerSRV = `package handler

import (
	"context"

	log "go-micro.dev/v5/logger"

	pb "{{.Dir}}/proto"
)

type {{title .Alias}} struct{}

// Return a new handler
func New() *{{title .Alias}} {
	return &{{title .Alias}}{}
}

// Call is a single request handler called via client.Call or the generated client code
func (e *{{title .Alias}}) Call(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	log.Info("Received {{title .Alias}}.Call request")
	rsp.Msg = "Hello " + req.Name
	return nil
}

// Stream is a server side stream handler called via client.Stream or the generated client code
func (e *{{title .Alias}}) Stream(ctx context.Context, req *pb.StreamingRequest, stream pb.{{title .Alias}}_StreamStream) error {
	log.Infof("Received {{title .Alias}}.Stream request with count: %d", req.Count)

	for i := 0; i < int(req.Count); i++ {
		log.Infof("Responding: %d", i)
		if err := stream.Send(&pb.StreamingResponse{
			Count: int64(i),
		}); err != nil {
			return err
		}
	}

	return nil
}
`

	SubscriberSRV = `package subscriber

import (
	"context"
	log "go-micro.dev/v5/logger"

	pb "{{.Dir}}/proto"
)

type {{title .Alias}} struct{}

func (e *{{title .Alias}}) Handle(ctx context.Context, msg *pb.Message) error {
	log.Info("Handler Received message: ", msg.Say)
	return nil
}

func Handler(ctx context.Context, msg *pb.Message) error {
	log.Info("Function Received message: ", msg.Say)
	return nil
}
`
)
