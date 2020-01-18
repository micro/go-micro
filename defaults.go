package micro

import (
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/store"

	// set defaults
	"github.com/micro/go-micro/broker/nats"
	gcli "github.com/micro/go-micro/client/grpc"
	gsrv "github.com/micro/go-micro/server/grpc"
	memStore "github.com/micro/go-micro/store/memory"
)

func init() {
	// default broker
	broker.DefaultBroker = nats.NewBroker(
		// embedded nats server
		nats.LocalServer(),
	)
	// new client initialisation
	client.NewClient = gcli.NewClient
	// new server initialisation
	server.NewServer = gsrv.NewServer
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
	// default store
	store.DefaultStore = memStore.NewStore()
}
