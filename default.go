package micro

import (
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/server"

	// set defaults
	gcli "github.com/asim/go-micro/plugins/client/grpc/v4"
	gsrv "github.com/asim/go-micro/plugins/server/grpc/v4"
)

func init() {
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
}
