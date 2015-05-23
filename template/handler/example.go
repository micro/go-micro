package handler

import (
	log "github.com/golang/glog"
	c "github.com/myodc/go-micro/context"
	"github.com/myodc/go-micro/server"
	example "github.com/myodc/go-micro/template/proto/example"

	"golang.org/x/net/context"
)

type Example struct{}

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {
	md, ok := c.GetMetaData(ctx)
	if ok {
		log.Infof("Received Example.Call request with metadata: %v", md)
	} else {
		log.Info("Received Example.Call request")
	}

	rsp.Msg = server.Id + ": Hello " + req.Name

	return nil
}
