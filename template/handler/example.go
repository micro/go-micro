package handler

import (
	"code.google.com/p/go.net/context"
	"code.google.com/p/goprotobuf/proto"

	"github.com/asim/go-micro/server"
	example "github.com/asim/go-micro/template/proto/example"
	log "github.com/golang/glog"
)

type Example struct{}

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {
	log.Info("Received Example.Call request")

	rsp.Msg = proto.String(server.Id + ": Hello " + req.GetName())

	return nil
}
