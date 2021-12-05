package micro

import (
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/server"

	// set defaults
	gcli "go-micro.dev/v4/plugins/client/grpc"
	gsrv "go-micro.dev/v4/plugins/server/grpc"
)

func init() {
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
}
