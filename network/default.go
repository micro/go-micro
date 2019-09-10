package network

import (
	"container/list"
	"errors"
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

var (
	// ErrMsgUnknown is returned when unknown message is attempted to send or receive
	ErrMsgUnknown = errors.New("unknown message")
	// ErrClientNotFound is returned when client for tunnel channel could not be found
	ErrClientNotFound = errors.New("client not found")
)

// node is network node
type node struct {
	sync.RWMutex
	// id is node id
	id string
	// address is node address
	address string
	// neighbours maps the node neighbourhood
	neighbours map[string]*node
	// network returns the node network
	network Network
	// lastSeen stores the time the node has been seen last time
	lastSeen time.Time
}

// Id is node ide
func (n *node) Id() string {
	return n.id
}

// Address returns node address
func (n *node) Address() string {
	return n.address
}

// Network returns node network
func (n *node) Network() Network {
	return n.network
}

// Neighbourhood returns node neighbourhood
func (n *node) Neighbourhood() []Node {
	var nodes []Node
	n.RLock()
	for _, neighbourNode := range n.neighbours {
		// make a copy of the node
		n := &node{
			id:      neighbourNode.id,
			address: neighbourNode.address,
			network: neighbourNode.network,
		}
		// NOTE: we do not care about neighbour's neighbours
		nodes = append(nodes, n)
	}
	n.RUnlock()

	return nodes
}

// network implements Network interface
type network struct {
	// node is network node
	*node
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

	// tunClient is a map of tunnel clients keyed over tunnel channel names
	tunClient map[string]transport.Client

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

	network := &network{
		node: &node{
			id:         options.Id,
			address:    options.Address,
			neighbours: make(map[string]*node),
		},
		options:   options,
		Router:    options.Router,
		Proxy:     options.Proxy,
		Tunnel:    options.Tunnel,
		server:    server,
		client:    client,
		tunClient: make(map[string]transport.Client),
	}

	network.node.network = network

	return network
}

// Options returns network options
func (n *network) Options() Options {
	n.Lock()
	options := n.options
	n.Unlock()

	return options
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
func (n *network) handleNetConn(sess tunnel.Session, msg chan *transport.Message) {
	for {
		m := new(transport.Message)
		if err := sess.Recv(m); err != nil {
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
func (n *network) processNetChan(client transport.Client, listener tunnel.Listener) {
	// receive network message queue
	recv := make(chan *transport.Message, 128)

	// accept NetworkChannel connections
	go n.acceptNetConn(listener, recv)

	for {
		select {
		case m := <-recv:
			// switch on type of message and take action
			switch m.Header["Micro-Method"] {
			case "connect":
				// mark the time the message has been received
				now := time.Now()
				pbNetConnect := &pbNet.Connect{}
				if err := proto.Unmarshal(m.Body, pbNetConnect); err != nil {
					log.Debugf("Network tunnel [%s] connect unmarshal error: %v", NetworkChannel, err)
					continue
				}
				// don't process your own messages
				if pbNetConnect.Node.Id == n.options.Id {
					continue
				}
				n.Lock()
				log.Debugf("Network received connect message from: %s", pbNetConnect.Node.Id)
				// if the entry already exists skip adding it
				if neighbour, ok := n.neighbours[pbNetConnect.Node.Id]; ok {
					// update lastSeen timestamp
					if n.neighbours[pbNetConnect.Node.Id].lastSeen.Before(now) {
						neighbour.lastSeen = now
					}
					n.Unlock()
					continue
				}
				// add a new neighbour
				// NOTE: new node does not have any neighbours
				n.neighbours[pbNetConnect.Node.Id] = &node{
					id:         pbNetConnect.Node.Id,
					address:    pbNetConnect.Node.Address,
					neighbours: make(map[string]*node),
					lastSeen:   now,
				}
				n.Unlock()
				// advertise yourself to the network
				if err := n.sendMsg("neighbour", NetworkChannel); err != nil {
					log.Debugf("Network failed to advertise neighbours: %v", err)
				}
				// advertise all the routes when a new node has connected
				if err := n.Router.Solicit(); err != nil {
					log.Debugf("Network failed to solicit routes: %s", err)
				}
			case "neighbour":
				// mark the time the message has been received
				now := time.Now()
				pbNetNeighbour := &pbNet.Neighbour{}
				if err := proto.Unmarshal(m.Body, pbNetNeighbour); err != nil {
					log.Debugf("Network tunnel [%s] neighbour unmarshal error: %v", NetworkChannel, err)
					continue
				}
				// don't process your own messages
				if pbNetNeighbour.Node.Id == n.options.Id {
					continue
				}
				n.Lock()
				log.Debugf("Network received neighbour message from: %s", pbNetNeighbour.Node.Id)
				// only add the neighbour if it is NOT already in node's list of neighbours
				_, exists := n.neighbours[pbNetNeighbour.Node.Id]
				if !exists {
					n.neighbours[pbNetNeighbour.Node.Id] = &node{
						id:         pbNetNeighbour.Node.Id,
						address:    pbNetNeighbour.Node.Address,
						neighbours: make(map[string]*node),
						lastSeen:   now,
					}
				}
				// update lastSeen timestamp
				if n.neighbours[pbNetNeighbour.Node.Id].lastSeen.Before(now) {
					n.neighbours[pbNetNeighbour.Node.Id].lastSeen = now
				}
				// update/store the neighbour node neighbours
				// NOTE: * we do NOT update lastSeen time for the neighbours of the neighbour
				//	 * even though we are NOT interested in neighbours of neighbours here
				// 	   we still allocate the map of neighbours for each of them
				for _, pbNeighbour := range pbNetNeighbour.Neighbours {
					neighbourNode := &node{
						id:         pbNeighbour.Id,
						address:    pbNeighbour.Address,
						neighbours: make(map[string]*node),
					}
					n.neighbours[pbNetNeighbour.Node.Id].neighbours[pbNeighbour.Id] = neighbourNode
				}
				n.Unlock()
				// send a solicit message when discovering a new node
				// NOTE: we need to send the solicit message here after the Lock is released as sendMsg locks, too
				if !exists {
					if err := n.sendMsg("solicit", ControlChannel); err != nil {
						log.Debugf("Network failed to send solicit message: %s", err)
					}
				}
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
				log.Debugf("Network received close message from: %s", pbNetClose.Node.Id)
				if err := n.pruneNode(pbNetClose.Node.Id); err != nil {
					log.Debugf("Network failed to prune the node %s: %v", pbNetClose.Node.Id, err)
					continue
				}
				n.Unlock()
			}
		case <-n.closed:
			return
		}
	}
}

// sendMsg sends a message to the tunnel channel
func (n *network) sendMsg(msgType string, channel string) error {
	node := &pbNet.Node{
		Id:      n.options.Id,
		Address: n.options.Address,
	}

	var protoMsg proto.Message

	switch msgType {
	case "connect":
		protoMsg = &pbNet.Connect{
			Node: node,
		}
	case "close":
		protoMsg = &pbNet.Close{
			Node: node,
		}
	case "solicit":
		protoMsg = &pbNet.Solicit{
			Node: node,
		}
	case "neighbour":
		n.RLock()
		nodes := make([]*pbNet.Node, len(n.neighbours))
		i := 0
		for id := range n.neighbours {
			nodes[i] = &pbNet.Node{
				Id:      id,
				Address: n.neighbours[id].address,
			}
			i++
		}
		n.RUnlock()
		protoMsg = &pbNet.Neighbour{
			Node:       node,
			Neighbours: nodes,
		}
	default:
		return ErrMsgUnknown
	}

	body, err := proto.Marshal(protoMsg)
	if err != nil {
		return err
	}
	// create transport message and chuck it down the pipe
	m := transport.Message{
		Header: map[string]string{
			"Micro-Method": msgType,
		},
		Body: body,
	}

	n.RLock()
	client, ok := n.tunClient[channel]
	if !ok {
		n.RUnlock()
		return ErrClientNotFound
	}
	n.RUnlock()

	log.Debugf("Network sending %s message from: %s", msgType, node.Id)
	if err := client.Send(&m); err != nil {
		return err
	}

	return nil
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
			// advertise yourself to the network
			if err := n.sendMsg("neighbour", NetworkChannel); err != nil {
				log.Debugf("Network failed to advertise neighbours: %v", err)
				continue
			}
		}
	}
}

// pruneNode removes a node with given id from the list of neighbours. It also removes all routes originted by this node.
// NOTE: this method is not thread-safe; when calling it make sure you lock the particular code segment
func (n *network) pruneNode(id string) error {
	delete(n.neighbours, id)
	// lookup all the routes originated at this node
	q := router.NewQuery(
		router.QueryRouter(id),
	)
	routes, err := n.Router.Table().Query(q)
	if err != nil && err != router.ErrRouteNotFound {
		return err
	}
	// delete the found routes
	log.Logf("Network deleting routes originated by router: %s", id)
	for _, route := range routes {
		if err := n.Router.Table().Delete(route); err != nil && err != router.ErrRouteNotFound {
			return err
		}
	}

	return nil
}

// prune the nodes that have not been seen for certain period of time defined by PruneTime
// Additionally, prune also removes all the routes originated by these nodes
func (n *network) prune() {
	prune := time.NewTicker(PruneTime)
	defer prune.Stop()

	for {
		select {
		case <-n.closed:
			return
		case <-prune.C:
			n.Lock()
			for id, node := range n.neighbours {
				if id == n.options.Id {
					continue
				}
				if time.Since(node.lastSeen) > PruneTime {
					log.Debugf("Network deleting node %s: reached prune time threshold", id)
					if err := n.pruneNode(id); err != nil {
						log.Debugf("Network failed to prune the node %s: %v", id, err)
						continue
					}
				}
			}
			n.Unlock()
		}
	}
}

// handleCtrlConn handles ControlChannel connections
func (n *network) handleCtrlConn(sess tunnel.Session, msg chan *transport.Message) {
	for {
		m := new(transport.Message)
		if err := sess.Recv(m); err != nil {
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
		for id := range node.neighbours {
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
func (n *network) processCtrlChan(client transport.Client, listener tunnel.Listener) {
	// receive control message queue
	recv := make(chan *transport.Message, 128)

	// accept ControlChannel cconnections
	go n.acceptCtrlConn(listener, recv)

	for {
		select {
		case m := <-recv:
			// switch on type of message and take action
			switch m.Header["Micro-Method"] {
			case "advert":
				now := time.Now()
				pbRtrAdvert := &pbRtr.Advert{}
				if err := proto.Unmarshal(m.Body, pbRtrAdvert); err != nil {
					log.Debugf("Network fail to unmarshal advert message: %v", err)
					continue
				}
				// don't process your own messages
				if pbRtrAdvert.Id == n.options.Id {
					continue
				}
				// loookup advertising node in our neighbourhood
				n.RLock()
				log.Debugf("Network received advert message from: %s", pbRtrAdvert.Id)
				advertNode, ok := n.neighbours[pbRtrAdvert.Id]
				if !ok {
					// advertising node has not been registered as our neighbour, yet
					// let's add it to the map of our neighbours
					advertNode = &node{
						id:         pbRtrAdvert.Id,
						neighbours: make(map[string]*node),
						lastSeen:   now,
					}
					n.neighbours[pbRtrAdvert.Id] = advertNode
					// send a solicit message when discovering a new node
					if err := n.sendMsg("solicit", NetworkChannel); err != nil {
						log.Debugf("Network failed to send solicit message: %s", err)
					}
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
						log.Debugf("Network route metric %d dropping node: %s", route.Metric, route.Router)
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
			case "solicit":
				pbNetSolicit := &pbNet.Solicit{}
				if err := proto.Unmarshal(m.Body, pbNetSolicit); err != nil {
					log.Debugf("Network fail to unmarshal solicit message: %v", err)
					continue
				}
				log.Debugf("Network received solicit message from: %s", pbNetSolicit.Node.Id)
				// don't process your own messages
				if pbNetSolicit.Node.Id == n.options.Id {
					continue
				}
				// advertise all the routes when a new node has connected
				if err := n.Router.Solicit(); err != nil {
					log.Debugf("Network failed to solicit routes: %s", err)
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

			log.Debugf("Network sending advert message from: %s", pbRtrAdvert.Id)
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
	ctrlClient, err := n.Tunnel.Dial(ControlChannel, tunnel.DialMulticast())
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
	netClient, err := n.Tunnel.Dial(NetworkChannel, tunnel.DialMulticast())
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
	n.Unlock()

	// send connect message to NetworkChannel
	// NOTE: in theory we could do this as soon as
	// Dial to NetworkChannel succeeds, but instead
	// we initialize all other node resources first
	if err := n.sendMsg("connect", NetworkChannel); err != nil {
		log.Debugf("Network failed to send connect message: %s", err)
	}

	// go resolving network nodes
	go n.resolve()
	// broadcast neighbourhood
	go n.announce(netClient)
	// prune stale nodes
	go n.prune()
	// listen to network messages
	go n.processNetChan(netClient, netListener)
	// advertise service routes
	go n.advertise(ctrlClient, advertChan)
	// accept and process routes
	go n.processCtrlChan(ctrlClient, ctrlListener)

	n.Lock()
	n.connected = true
	n.Unlock()

	return nil
}

// Nodes returns a list of all network nodes
func (n *network) Nodes() []Node {
	//track the visited nodes
	visited := make(map[string]*node)
	// queue of the nodes to visit
	queue := list.New()

	// we need to freeze the network graph here
	// otherwise we might get invalid results
	n.RLock()
	defer n.RUnlock()

	// push network node to the back of queue
	queue.PushBack(n.node)
	// mark the node as visited
	visited[n.node.id] = n.node

	// keep iterating over the queue until its empty
	for queue.Len() > 0 {
		// pop the node from the front of the queue
		qnode := queue.Front()
		// iterate through all of its neighbours
		// mark the visited nodes; enqueue the non-visted
		for id, node := range qnode.Value.(*node).neighbours {
			if _, ok := visited[id]; !ok {
				visited[id] = node
				queue.PushBack(node)
			}
		}
		// remove the node from the queue
		queue.Remove(qnode)
	}

	var nodes []Node
	// collect all the nodes and return them
	for _, node := range visited {
		nodes = append(nodes, node)
	}

	return nodes
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
	// lock this operation
	n.Lock()

	if !n.connected {
		n.Unlock()
		return nil
	}

	select {
	case <-n.closed:
		n.Unlock()
		return nil
	default:
		// TODO: send close message to the network channel
		close(n.closed)
		// set connected to false
		n.connected = false

		// unlock the lock otherwise we'll deadlock sending the close
		n.Unlock()

		// send close message only if we managed to connect to NetworkChannel
		log.Debugf("Sending close message from: %s", n.options.Id)
		if err := n.sendMsg("close", NetworkChannel); err != nil {
			log.Debugf("Network failed to send close message: %s", err)
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
