package network

import (
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/router"
	pb "github.com/micro/go-micro/router/proto"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/tunnel"
	tr "github.com/micro/go-micro/tunnel/transport"
	"github.com/micro/go-micro/util/log"
)

var (
	// ControlChannel is the name of the tunnel channel for passing contron message
	ControlChannel = "control-msg"
)

// network implements Network interface
type network struct {
	// options configure the network
	// TODO: we might end up embedding
	options Options
	// rtr is network router
	router.Router
	// prx is network proxy
	proxy.Proxy
	// tun is network tunnel
	tunnel.Tunnel
	// srv is network server
	srv server.Server
	// client is network client
	client client.Client

	sync.RWMutex
	// connected marks the network as connected
	connected bool
	// closed closes the network
	closed chan bool
}

// newNetwork returns a new network node
func newNetwork(opts ...Option) Network {
	options := DefaultOptions()

	for _, o := range opts {
		o(&options)
	}

	// init tunnel address to the network bind address
	options.Tunnel.Init(
		tunnel.Address(options.Address),
	)

	// create tunnel client with tunnel transport
	tunTransport := transport.NewTransport(
		tr.WithTunnel(options.Tunnel),
	)

	// srv is network server
	srv := server.NewServer(
		server.Transport(tunTransport),
	)

	// client is network client
	client := client.NewClient(
		client.Transport(tunTransport),
	)

	return &network{
		options: options,
		Router:  options.Router,
		Proxy:   options.Proxy,
		Tunnel:  options.Tunnel,
		srv:     srv,
		client:  client,
	}
}

// Name returns network name
func (n *network) Name() string {
	return n.options.Name
}

// Address returns network bind address
func (n *network) Address() string {
	return n.options.Address
}

func (n *network) resolveNodes() ([]string, error) {
	// resolve the network address to network nodes
	records, err := n.options.Resolver.Resolve(n.options.Name)
	if err != nil {
		return nil, err
	}

	// collect network node addresses
	nodes := make([]string, len(records))
	for i, record := range records {
		nodes[i] = record.Address
	}

	return nodes, nil
}

func (n *network) resolve() {
	resolve := time.NewTicker(ResolveTime)
	defer resolve.Stop()

	for {
		select {
		case <-n.closed:
			return
		case <-resolve.C:
			nodes, err := n.resolveNodes()
			if err != nil {
				log.Debugf("Network failed to resolve nodes: %v", err)
				continue
			}
			// initialize the tunnel
			n.Tunnel.Init(
				tunnel.Nodes(nodes...),
			)
		}
	}
}

func (n *network) process(client transport.Client) {
	for {
		m := new(transport.Message)
		if err := client.Recv(m); err != nil {
			// TODO: should we bail here?
			log.Debugf("Network advert receive error: %v", err)
			return
		}

		// switch on type of message and take action
		switch m.Header["Micro-Tunnel"] {
		case n.Router.Options().Id:
			// NOTE: this should not happen
			// skip local adverts
			continue
		default:
			pbAdvert := &pb.Advert{}
			if err := proto.Unmarshal(m.Body, pbAdvert); err != nil {
				continue
			}

			var events []*router.Event
			for _, event := range pbAdvert.Events {
				route := router.Route{
					Service: event.Route.Service,
					Address: event.Route.Address,
					Gateway: event.Route.Gateway,
					Network: event.Route.Network,
					Link:    event.Route.Link,
					Metric:  int(event.Route.Metric),
				}
				e := &router.Event{
					Type:      router.EventType(event.Type),
					Timestamp: time.Unix(0, pbAdvert.Timestamp),
					Route:     route,
				}
				events = append(events, e)
			}
			advert := &router.Advert{
				Id:        pbAdvert.Id,
				Type:      router.AdvertType(pbAdvert.Type),
				Timestamp: time.Unix(0, pbAdvert.Timestamp),
				TTL:       time.Duration(pbAdvert.Ttl),
				Events:    events,
			}

			if err := n.Router.Process(advert); err != nil {
				log.Debugf("Network failed to process advert %s: %v", advert.Id, err)
				continue
			}
		}
	}
}

// advertise advertises routes to the network
func (n *network) advertise(client transport.Client, advertChan <-chan *router.Advert) {
	for {
		select {
		// process local adverts and randomly fire them at other nodes
		case advert := <-advertChan:
			// create a proto advert
			var events []*pb.Event
			for _, event := range advert.Events {
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
			pbAdvert := &pb.Advert{
				Id:        advert.Id,
				Type:      pb.AdvertType(advert.Type),
				Timestamp: advert.Timestamp.UnixNano(),
				Events:    events,
			}
			body, err := proto.Marshal(pbAdvert)
			if err != nil {
				// TODO: should we bail here?
				log.Debugf("Network failed to marshal message: %v", err)
				continue
			}
			// create transport message and chuck it down the pipe
			m := transport.Message{
				Header: map[string]string{
					"Micro-Method": "advert",
				},
				Body: body,
			}
			if err := client.Send(&m); err != nil {
				log.Debugf("Network failed to send advert %s: %v", pbAdvert.Id, err)
				continue
			}
		case <-n.closed:
			return
		}
	}
}

// Connect connects the network
func (n *network) Connect() error {
	n.Lock()
	defer n.Unlock()

	// return if already connected
	if n.connected {
		return nil
	}

	// try to resolve network nodes
	nodes, err := n.resolveNodes()
	if err != nil {
		return err
	}

	// connect network tunnel
	if err := n.Tunnel.Connect(); err != nil {
		return err
	}

	// initialize the tunnel to resolved nodes
	n.Tunnel.Init(
		tunnel.Nodes(nodes...),
	)

	// dial into ControlChannel to send route adverts
	client, err := n.Tunnel.Dial(ControlChannel)
	if err != nil {
		// TODO: should we stop the tunnel here?
		return err
	}

	// create closed channel
	n.closed = make(chan bool)

	// keep resolving network nodes
	go n.resolve()

	// TODO: do we assume the router has been started?
	// start advertising routes
	advertChan, err := n.options.Router.Advertise()
	if err != nil {
		return err
	}

	// advertise routes
	go n.advertise(client, advertChan)
	// process routes
	go n.process(client)

	// set connected to true
	n.connected = true

	return nil
}

func (n *network) close() error {
	// stop the router
	if err := n.Router.Stop(); err != nil {
		return err
	}

	// close the tunnel
	if err := n.Tunnel.Close(); err != nil {
		return err
	}

	return nil
}

// Close closes network connection
func (n *network) Close() error {
	n.Lock()
	defer n.Unlock()

	if !n.connected {
		return nil
	}

	select {
	case <-n.closed:
		return nil
	default:
		close(n.closed)
		// set connected to false
		n.connected = false
	}

	return n.close()
}

// Client returns network client
func (n *network) Client() client.Client {
	return n.client
}

// Server returns network server
func (n *network) Server() server.Server {
	return n.srv
}
