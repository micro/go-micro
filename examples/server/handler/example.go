package handler

import (
	log "github.com/golang/glog"
	c "github.com/kynrai/go-micro/context"
	example "github.com/kynrai/go-micro/examples/server/proto/example"
	"github.com/kynrai/go-micro/server"

	"golang.org/x/net/context"
)

type Example struct{}

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {
	md, _ := c.GetMetadata(ctx)
	log.Infof("Received Example.Call request with metadata: %v", md)
	rsp.Msg = server.Config().Id() + ": Hello " + req.Name
	return nil
}

func (e *Example) Stream(ctx context.Context, req *example.StreamingRequest, response func(interface{}) error) error {
	log.Infof("Received Example.Stream request with count: %d", req.Count)
	for i := 0; i < int(req.Count); i++ {
		log.Infof("Responding: %d", i)

		r := &example.StreamingResponse{
			Count: int64(i),
		}

		if err := response(r); err != nil {
			return err
		}
	}

	return nil
}
