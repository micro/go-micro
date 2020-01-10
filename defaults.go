package micro

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/store"

	// set defaults
	gcli "github.com/micro/go-micro/client/grpc"
	gsrv "github.com/micro/go-micro/server/grpc"
	memStore "github.com/micro/go-micro/store/memory"
)

func init() {
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
	// default store
	store.DefaultStore = memStore.NewStore()
}
