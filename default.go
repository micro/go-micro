package micro

import (
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/server"

	// set defaults
	gcli "github.com/asim/go-micro/plugins/client/grpc/v3"
	gsrv "github.com/asim/go-micro/plugins/server/grpc/v3"
)

func init() {
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
}
