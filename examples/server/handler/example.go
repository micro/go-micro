package handler

import (
	log "github.com/golang/glog"
	c "github.com/myodc/go-micro/context"
	example "github.com/myodc/go-micro/examples/server/proto/example"
	"github.com/myodc/go-micro/server"

	"golang.org/x/net/context"
)

type Example struct{}

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {
	md, _ := c.GetMetadata(ctx)
	log.Info("Received Example.Call request with metadata: %v", md)
	rsp.Msg = server.Config().Id() + ": Hello " + req.Name
	return nil
}
