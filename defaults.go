package micro

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"

	// set defaults
	gcli "github.com/micro/go-micro/client/grpc"
	gsrv "github.com/micro/go-micro/server/grpc"
)

func init() {
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
}
