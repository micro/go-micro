package network

import (
	"context"
	"sync"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/network/proxy/mucp"
	"github.com/micro/go-micro/network/router"
	"github.com/micro/go-micro/network/router/handler"
	pb "github.com/micro/go-micro/network/router/proto"
	"github.com/micro/go-micro/server"
)

type network struct {
	options Options
	handler server.Router
	router  pb.RouterService

	sync.RWMutex
	connected bool
	exit      chan bool
}

// process processes router advertisements and randomly sends the advert
// to a node in the network. Over a period of time the routers should converge.
func (n *network) process(advertChan <-chan *router.Advert) {
	for {
		select {
		// process local adverts and randomly fire them at other nodes
		case a := <-advertChan:
			// create a proto advert
			var events []*pb.Event
			for _, event := range a.Events {
				route := &pb.Route{
					Service: event.Route.Service,
					Address: event.Route.Address,
					Gateway: event.Route.Gateway,
					Network: event.Route.Network,
					Link:    event.Route.Link,
					Metric:  int64(event.Route.Metric),
				}
				e := &pb.Event{
					Type:      pb.EventType(event.Type),
					Timestamp: event.Timestamp.UnixNano(),
					Route:     route,
				}
				events = append(events, e)
			}

			// fire the advert to a random network node
			n.router.Process(context.Background(), &pb.Advert{
				Id:        n.options.Router.Options().Id,
				Type:      pb.AdvertType(a.Type),
				Timestamp: a.Timestamp.UnixNano(),
				Events:    events,
			})
		case <-n.exit:
			return
		}
	}
}
func (n *network) Name() string {
	return n.options.Name
}

// Implements the server.ServeRequest method.
func (n *network) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// If we're being called then execute our handlers
	if req.Service() == n.options.Name {
		return n.handler.ServeRequest(ctx, req, rsp)
	}

	// execute the proxy
	return n.options.Proxy.ServeRequest(ctx, req, rsp)
}

func (n *network) Connect() error {
	n.Lock()
	defer n.Unlock()

	// check if we're connected
	if n.connected {
		return nil
	}

	n.exit = make(chan bool)

	// start advertising
	advertChan, err := n.options.Router.Advertise()
	if err != nil {
		return err
	}

	// process the adverts
	go n.process(advertChan)

	// start the server
	if err := n.options.Server.Start(); err != nil {
		return err
	}

	// set connected to true
	n.connected = true

	return nil
}

func (n *network) Close() error {
	n.Lock()
	defer n.Unlock()

	// check if we're connected
	if !n.connected {
		return nil
	}

	close(n.exit)

	// set connected to false
	n.connected = false

	// stop the router
	if err := n.options.Router.Stop(); err != nil {
		return err
	}

	// stop the server
	return n.options.Server.Stop()
}

// newNetwork returns a new network node
func newNetwork(opts ...Option) Network {
	options := Options{
		Name:    DefaultName,
		Address: DefaultAddress,
		Client:  client.DefaultClient,
		Server:  server.DefaultServer,
		Proxy:   mucp.NewProxy(),
		Router:  router.DefaultRouter,
	}

	for _, o := range opts {
		o(&options)
	}

	// get the default server handler
	sr := server.DefaultRouter
	// create new router handler
	hd := sr.NewHandler(&handler.Router{options.Router})
	// register the router handler
	sr.Handle(hd)

	// set the server name
	options.Server.Init(
		server.Name(options.Name),
		server.Address(options.Address),
		server.Advertise(options.Advertise),
		server.WithRouter(sr),
	)

	return &network{
		options: options,
		handler: sr,
		router:  pb.NewRouterService(options.Name, options.Client),
	}
}
