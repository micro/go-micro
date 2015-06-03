package server

import (
	"github.com/myodc/go-micro/server/proto/health"
	"golang.org/x/net/context"
)

type Debug struct{}

func (d *Debug) Health(ctx context.Context, req *health.Request, rsp *health.Response) error {
	rsp.Status = "ok"
	return nil
}

func registerHealthChecker(s Server) {
	s.Handle(s.NewHandler(&Debug{}))
}
