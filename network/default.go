package network

import (
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/client"
	rtr "github.com/micro/go-micro/client/selector/router"
	pbNet "github.com/micro/go-micro/network/proto"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/router"
	pbRtr "github.com/micro/go-micro/router/proto"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/tunnel"
	tun "github.com/micro/go-micro/tunnel/transport"
	"github.com/micro/go-micro/util/log"
)

var (
	// NetworkChannel is the name of the tunnel channel for passing network messages
	NetworkChannel = "network"
	// ControlChannel is the name of the tunnel channel for passing control message
	ControlChannel = "control"
	// DefaultLink is default network link
	DefaultLink = "network"
)

// node is network node
type node struct {
	// id is node id
	id string
	// address is node address
	address string
	// neighbours are node neightbours
	neighbours map[string]*node
}

// network implements Network interface
type network struct {
	// options configure the network
	options Options
	// rtr is network router
	router.Router
	// prx is network proxy
	proxy.Proxy
	// tun is network tunnel
	tunnel.Tunnel
	// server is network server
	server server.Server
	// client is network client
	client client.Client

	// tunClient is a mao of tunnel clients keyed over channel names
	tunClient map[string]transport.Client

	sync.RWMutex
	// connected marks the network as connected
	connected bool
	// closed closes the network
	closed chan bool
	// neighbours maps the node neighbourhood
	neighbours map[string]*node
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
		tunnel.Nodes(options.Nodes...),
	)

	// init router Id to the network id
	options.Router.Init(
		router.Id(options.Id),
	)

	// create tunnel client with tunnel transport
	tunTransport := tun.NewTransport(
		tun.WithTunnel(options.Tunnel),
	)

	// server is network server
	server := server.NewServer(
		server.Id(options.Id),
		server.Address(options.Address),
		server.Name(options.Name),
		server.Transport(tunTransport),
	)

	// client is network client
	client := client.NewClient(
		client.Transport(tunTransport),
		client.Selector(
			rtr.NewSelector(
				rtr.WithRouter(options.Router),
			),
		),
	)

	return &network{
		options:    options,
		Router:     options.Router,
		Proxy:      options.Proxy,
		Tunnel:     options.Tunnel,
		server:     server,
		client:     client,
		tunClient:  make(map[string]transport.Client),
		neighbours: make(map[string]*node),
	}
}

// Name returns network name
func (n *network) Name() string {
	return n.options.Name
}

// Address returns network bind address
func (n *network) Address() string {
	return n.Tunnel.Address()
}

// resolveNodes resolves network nodes to addresses
func (n *network) resolveNodes() ([]string, error) {
	// resolve the network address to network nodes
	records, err := n.options.Resolver.Resolve(n.options.Name)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string]bool)

	// collect network node addresses
	var nodes []string
	for _, record := range records {
		nodes = append(nodes, record.Address)
		nodeMap[record.Address] = true
	}

	// append seed nodes if we have them
	for _, node := range n.options.Nodes {
		if _, ok := nodeMap[node]; !ok {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// resolve continuously resolves network nodes and initializes network tunnel with resolved addresses
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

// handleNetConn handles network announcement messages
func (n *network) handleNetConn(conn tunnel.Conn, msg chan *transport.Message) {
	for {
		m := new(transport.Message)
		if err := conn.Recv(m); err != nil {
			// TODO: should we bail here?
			log.Debugf("Network tunnel [%s] receive error: %v", NetworkChannel, err)
			return
		}

		select {
		case msg <- m:
		case <-n.closed:
			return
		}
	}
}

// acceptNetConn accepts connections from NetworkChannel
func (n *network) acceptNetConn(l tunnel.Listener, recv chan *transport.Message) {
	for {
		// accept a connection
		conn, err := l.Accept()
		if err != nil {
			// TODO: handle this
			log.Debugf("Network tunnel [%s] accept error: %v", NetworkChannel, err)
			return
		}

		select {
		case <-n.closed:
			return
		default:
			// go handle NetworkChannel connection
			go n.handleNetConn(conn, recv)
		}
	}
}

// processNetChan processes messages received on NetworkChannel
func (n *network) processNetChan(l tunnel.Listener) {
	// receive network message queue
	recv := make(chan *transport.Message, 128)

	// accept NetworkChannel connections
	go n.acceptNetConn(l, recv)

	for {
		select {
		case m := <-recv:
			// switch on type of message and take action
			switch m.Header["Micro-Method"] {
			case "connect":
				pbNetConnect := &pbNet.Connect{}
				if err := proto.Unmarshal(m.Body, pbNetConnect); err != nil {
					log.Debugf("Network tunnel [%s] connect unmarshal error: %v", NetworkChannel, err)
					continue
				}
				// don't process your own messages
				if pbNetConnect.Node.Id == n.options.Id {
					continue
				}
				neighbour := &node{
					id:         pbNetConnect.Node.Id,
					address:    pbNetConnect.Node.Address,
					neighbours: make(map[string]*node),
				}
				n.Lock()
				n.neighbours[neighbour.id] = neighbour
				n.Unlock()
			case "neighbour":
				pbNetNeighbour := &pbNet.Neighbour{}
				if err := proto.Unmarshal(m.Body, pbNetNeighbour); err != nil {
					log.Debugf("Network tunnel [%s] neighbour unmarshal error: %v", NetworkChannel, err)
					continue
				}
				// don't process your own messages
				if pbNetNeighbour.Node.Id == n.options.Id {
					continue
				}
				neighbour := &node{
					id:         pbNetNeighbour.Node.Id,
					address:    pbNetNeighbour.Node.Address,
					neighbours: make(map[string]*node),
				}
				n.Lock()
				// we override the existing neighbour map
				n.neighbours[neighbour.id] = neighbour
				// store the neighbouring node and its neighbours
				for _, pbNeighbour := range pbNetNeighbour.Neighbours {
					neighbourNode := &node{
						id:      pbNeighbour.Id,
						address: pbNeighbour.Address,
					}
					n.neighbours[neighbour.id].neighbours[neighbourNode.id] = neighbourNode
				}
				n.Unlock()
			case "close":
				pbNetClose := &pbNet.Close{}
				if err := proto.Unmarshal(m.Body, pbNetClose); err != nil {
					log.Debugf("Network tunnel [%s] close unmarshal error: %v", NetworkChannel, err)
					continue
				}
				// don't process your own messages
				if pbNetClose.Node.Id == n.options.Id {
					continue
				}
				n.Lock()
				delete(n.neighbours, pbNetClose.Node.Id)
				n.Unlock()
			}
		case <-n.closed:
			return
		}
	}
}

// announce announces node neighbourhood to the network
func (n *network) announce(client transport.Client) {
	announce := time.NewTicker(AnnounceTime)
	defer announce.Stop()

	for {
		select {
		case <-n.closed:
			return
		case <-announce.C:
			n.RLock()
			nodes := make([]*pbNet.Node, len(n.neighbours))
			i := 0
			for id, _ := range n.neighbours {
				nodes[i] = &pbNet.Node{
					Id:      id,
					Address: n.neighbours[id].address,
				}
				i++
			}
			n.RUnlock()

			node := &pbNet.Node{
				Id:      n.options.Id,
				Address: n.options.Address,
			}
			pbNetNeighbour := &pbNet.Neighbour{
				Node:       node,
				Neighbours: nodes,
			}

			body, err := proto.Marshal(pbNetNeighbour)
			if err != nil {
				// TODO: should we bail here?
				log.Debugf("Network failed to marshal neighbour message: %v", err)
				continue
			}
			// create transport message and chuck it down the pipe
			m := transport.Message{
				Header: map[string]string{
					"Micro-Method": "neighbour",
				},
				Body: body,
			}

			if err := client.Send(&m); err != nil {
				log.Debugf("Network failed to send neighbour messsage: %v", err)
				continue
			}
		}
	}
}

// handleCtrlConn handles ControlChannel connections
func (n *network) handleCtrlConn(conn tunnel.Conn, msg chan *transport.Message) {
	for {
		m := new(transport.Message)
		if err := conn.Recv(m); err != nil {
			// TODO: should we bail here?
			log.Debugf("Network tunnel advert receive error: %v", err)
			return
		}

		select {
		case msg <- m:
		case <-n.closed:
			return
		}
	}
}

// acceptCtrlConn accepts connections from ControlChannel
func (n *network) acceptCtrlConn(l tunnel.Listener, recv chan *transport.Message) {
	for {
		// accept a connection
		conn, err := l.Accept()
		if err != nil {
			// TODO: handle this
			log.Debugf("Network tunnel [%s] accept error: %v", ControlChannel, err)
			return
		}

		select {
		case <-n.closed:
			return
		default:
			// go handle ControlChannel connection
			go n.handleCtrlConn(conn, recv)
		}
	}
}

// setRouteMetric calculates metric of the route and updates it in place
// - Local route metric is 1
// - Routes with ID of adjacent neighbour are 10
// - Routes of neighbours of the advertiser are 100
// - Routes beyond your neighbourhood are 1000
func (n *network) setRouteMetric(route *router.Route) {
	// we are the origin of the route
	if route.Router == n.options.Id {
		route.Metric = 1
		return
	}

	n.RLock()
	// check if the route origin is our neighbour
	if _, ok := n.neighbours[route.Router]; ok {
		route.Metric = 10
		n.RUnlock()
		return
	}

	// check if the route origin is the neighbour of our neighbour
	for _, node := range n.neighbours {
		for id, _ := range node.neighbours {
			if route.Router == id {
				route.Metric = 100
				n.RUnlock()
				return
			}
		}
	}
	n.RUnlock()

	// the origin of the route is beyond our neighbourhood
	route.Metric = 1000
}

// processCtrlChan processes messages received on ControlChannel
func (n *network) processCtrlChan(l tunnel.Listener) {
	// receive control message queue
	recv := make(chan *transport.Message, 128)

	// accept ControlChannel cconnections
	go n.acceptCtrlConn(l, recv)

	for {
		select {
		case m := <-recv:
			// switch on type of message and take action
			switch m.Header["Micro-Method"] {
			case "advert":
				pbRtrAdvert := &pbRtr.Advert{}
				if err := proto.Unmarshal(m.Body, pbRtrAdvert); err != nil {
					log.Debugf("Network fail to unmarshal advert message: %v", err)
					continue
				}

				// loookup advertising node in our neighbourhood
				n.RLock()
				advertNode, ok := n.neighbours[pbRtrAdvert.Id]
				if !ok {
					// advertising node has not been registered as our neighbour, yet
					// let's add it to the map of our neighbours
					advertNode = &node{
						id:         pbRtrAdvert.Id,
						neighbours: make(map[string]*node),
					}
					n.neighbours[pbRtrAdvert.Id] = advertNode
				}
				n.RUnlock()

				var events []*router.Event
				for _, event := range pbRtrAdvert.Events {
					// set the address of the advertising node
					// we know Route.Gateway is the address of advertNode
					// NOTE: this is true only when advertNode had not been registered
					// as our neighbour when we received the advert from it
					if advertNode.address == "" {
						advertNode.address = event.Route.Gateway
					}
					// if advertising node id is not the same as Route.Router
					// we know the advertising node is not the origin of the route
					if advertNode.id != event.Route.Router {
						// if the origin router is not in the advertising node neighbourhood
						// we can't rule out potential routing loops so we bail here
						if _, ok := advertNode.neighbours[event.Route.Router]; !ok {
							continue
						}
					}
					route := router.Route{
						Service: event.Route.Service,
						Address: event.Route.Address,
						Gateway: event.Route.Gateway,
						Network: event.Route.Network,
						Router:  event.Route.Router,
						Link:    event.Route.Link,
						Metric:  int(event.Route.Metric),
					}
					// set the route metric
					n.setRouteMetric(&route)
					// throw away metric bigger than 1000
					if route.Metric > 1000 {
						continue
					}
					// create router event
					e := &router.Event{
						Type:      router.EventType(event.Type),
						Timestamp: time.Unix(0, pbRtrAdvert.Timestamp),
						Route:     route,
					}
					events = append(events, e)
				}
				advert := &router.Advert{
					Id:        pbRtrAdvert.Id,
					Type:      router.AdvertType(pbRtrAdvert.Type),
					Timestamp: time.Unix(0, pbRtrAdvert.Timestamp),
					TTL:       time.Duration(pbRtrAdvert.Ttl),
					Events:    events,
				}

				if err := n.Router.Process(advert); err != nil {
					log.Debugf("Network failed to process advert %s: %v", advert.Id, err)
					continue
				}
			}
		case <-n.closed:
			return
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
			var events []*pbRtr.Event
			for _, event := range advert.Events {
				// NOTE: we override the Gateway and Link fields here
				route := &pbRtr.Route{
					Service: event.Route.Service,
					Address: event.Route.Address,
					Gateway: n.options.Address,
					Network: event.Route.Network,
					Router:  event.Route.Router,
					Link:    DefaultLink,
					Metric:  int64(event.Route.Metric),
				}
				e := &pbRtr.Event{
					Type:      pbRtr.EventType(event.Type),
					Timestamp: event.Timestamp.UnixNano(),
					Route:     route,
				}
				events = append(events, e)
			}
			pbRtrAdvert := &pbRtr.Advert{
				Id:        advert.Id,
				Type:      pbRtr.AdvertType(advert.Type),
				Timestamp: advert.Timestamp.UnixNano(),
				Events:    events,
			}
			body, err := proto.Marshal(pbRtrAdvert)
			if err != nil {
				// TODO: should we bail here?
				log.Debugf("Network failed to marshal advert message: %v", err)
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
				log.Debugf("Network failed to send advert %s: %v", pbRtrAdvert.Id, err)
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
		log.Debugf("Network failed to resolve nodes: %v", err)
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
	ctrlClient, err := n.Tunnel.Dial(ControlChannel)
	if err != nil {
		return err
	}

	n.tunClient[ControlChannel] = ctrlClient

	// listen on ControlChannel
	ctrlListener, err := n.Tunnel.Listen(ControlChannel)
	if err != nil {
		return err
	}

	// dial into NetworkChannel to send network messages
	netClient, err := n.Tunnel.Dial(NetworkChannel)
	if err != nil {
		return err
	}

	n.tunClient[NetworkChannel] = netClient

	// listen on NetworkChannel
	netListener, err := n.Tunnel.Listen(NetworkChannel)
	if err != nil {
		return err
	}

	// create closed channel
	n.closed = make(chan bool)

	// start the router
	if err := n.options.Router.Start(); err != nil {
		return err
	}

	// start advertising routes
	advertChan, err := n.options.Router.Advertise()
	if err != nil {
		return err
	}

	// start the server
	if err := n.server.Start(); err != nil {
		return err
	}

	// go resolving network nodes
	go n.resolve()
	// broadcast neighbourhood
	go n.announce(netClient)
	// listen to network messages
	go n.processNetChan(netListener)
	// advertise service routes
	go n.advertise(ctrlClient, advertChan)
	// accept and process routes
	go n.processCtrlChan(ctrlListener)

	// set connected to true
	n.connected = true

	// send connect message to NetworkChannel
	node := &pbNet.Node{
		Id:      n.options.Id,
		Address: n.options.Address,
	}
	pbNetConnect := &pbNet.Connect{
		Node: node,
	}

	// only proceed with sending to NetworkChannel if marshal succeeds
	if body, err := proto.Marshal(pbNetConnect); err == nil {
		m := transport.Message{
			Header: map[string]string{
				"Micro-Method": "connect",
			},
			Body: body,
		}

		if err := netClient.Send(&m); err != nil {
			log.Debugf("Network failed to send connect messsage: %v", err)
		}
	}

	return nil
}

func (n *network) close() error {
	// stop the server
	if err := n.server.Stop(); err != nil {
		return err
	}

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
		// TODO: send close message to the network channel
		close(n.closed)
		// set connected to false
		n.connected = false
	}

	// send close message only if we managed to connect to NetworkChannel
	if netClient, ok := n.tunClient[NetworkChannel]; ok {
		// send connect message to NetworkChannel
		node := &pbNet.Node{
			Id:      n.options.Id,
			Address: n.options.Address,
		}
		pbNetClose := &pbNet.Close{
			Node: node,
		}

		// only proceed with sending to NetworkChannel if marshal succeeds
		if body, err := proto.Marshal(pbNetClose); err == nil {
			// create transport message and chuck it down the pipe
			m := transport.Message{
				Header: map[string]string{
					"Micro-Method": "close",
				},
				Body: body,
			}

			if err := netClient.Send(&m); err != nil {
				log.Debugf("Network failed to send close messsage: %v", err)
			}
		}
	}

	return n.close()
}

// Client returns network client
func (n *network) Client() client.Client {
	return n.client
}

// Server returns network server
func (n *network) Server() server.Server {
	return n.server
}
