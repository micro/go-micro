package network

import (
	"errors"
	"fmt"
	"hash/fnv"
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
	// ErrClientNotFound is returned when client for tunnel channel could not be found
	ErrClientNotFound = errors.New("client not found")
)

// network implements Network interface
type network struct {
	// node is network node
	*node
	// options configure the network
	options Options
	// rtr is network router
	router router.Router
	// prx is network proxy
	proxy proxy.Proxy
	// tun is network tunnel
	tunnel tunnel.Tunnel
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
		tunnel.Nodes(options.Peers...),
	)

	// init router Id to the network id
	options.Router.Init(
		router.Id(options.Id),
	)

	// create tunnel client with tunnel transport
	tunTransport := tun.NewTransport(
		tun.WithTunnel(options.Tunnel),
	)

	// set the address to advertise
	address := options.Address
	if len(options.Advertise) > 0 {
		address = options.Advertise
	}

	// server is network server
	server := server.NewServer(
		server.Id(options.Id),
		server.Address(options.Id),
		server.Advertise(address),
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
			id:      options.Id,
			address: address,
			peers:   make(map[string]*node),
		},
		options:   options,
		router:    options.Router,
		proxy:     options.Proxy,
		tunnel:    options.Tunnel,
		server:    server,
		client:    client,
		tunClient: make(map[string]transport.Client),
	}

	network.node.network = network

	return network
}

// Options returns network options
func (n *network) Options() Options {
	n.RLock()
	defer n.RUnlock()

	options := n.options

	return options
}

// Name returns network name
func (n *network) Name() string {
	return n.options.Name
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
	for _, node := range n.options.Peers {
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
			n.tunnel.Init(
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
				log.Debugf("Network received connect message from: %s", pbNetConnect.Node.Id)
				peer := &node{
					id:       pbNetConnect.Node.Id,
					address:  pbNetConnect.Node.Address,
					peers:    make(map[string]*node),
					lastSeen: now,
				}
				if err := n.node.AddPeer(peer); err == ErrPeerExists {
					log.Debugf("Network peer exists, refreshing: %s", peer.id)
					// update lastSeen time for the existing node
					if err := n.RefreshPeer(peer.id, now); err != nil {
						log.Debugf("Network failed refreshing peer %s: %v", peer.id, err)
					}
					continue
				}
				// get node peers down to MaxDepth encoded in protobuf
				msg := PeersToProto(n.node, MaxDepth)
				// advertise yourself to the network
				if err := n.sendMsg("peer", msg, NetworkChannel); err != nil {
					log.Debugf("Network failed to advertise peers: %v", err)
				}
				// advertise all the routes when a new node has connected
				if err := n.router.Solicit(); err != nil {
					log.Debugf("Network failed to solicit routes: %s", err)
				}
			case "peer":
				// mark the time the message has been received
				now := time.Now()
				pbNetPeer := &pbNet.Peer{}
				if err := proto.Unmarshal(m.Body, pbNetPeer); err != nil {
					log.Debugf("Network tunnel [%s] peer unmarshal error: %v", NetworkChannel, err)
					continue
				}
				// don't process your own messages
				if pbNetPeer.Node.Id == n.options.Id {
					continue
				}
				log.Debugf("Network received peer message from: %s", pbNetPeer.Node.Id)
				peer := &node{
					id:       pbNetPeer.Node.Id,
					address:  pbNetPeer.Node.Address,
					peers:    make(map[string]*node),
					lastSeen: now,
				}
				if err := n.node.AddPeer(peer); err == nil {
					// send a solicit message when discovering new peer
					msg := &pbRtr.Solicit{
						Id: n.options.Id,
					}
					if err := n.sendMsg("solicit", msg, ControlChannel); err != nil {
						log.Debugf("Network failed to send solicit message: %s", err)
					}
					continue
					// we're expecting any error to be ErrPeerExists
				} else if err != ErrPeerExists {
					log.Debugf("Network got error adding peer %v", err)
					continue
				}

				log.Debugf("Network peer exists, refreshing: %s", pbNetPeer.Node.Id)
				// update lastSeen time for the peer
				if err := n.RefreshPeer(pbNetPeer.Node.Id, now); err != nil {
					log.Debugf("Network failed refreshing peer %s: %v", pbNetPeer.Node.Id, err)
				}

				// NOTE: we don't unpack MaxDepth toplogy
				peer = UnpackPeerTopology(pbNetPeer, now, MaxDepth-1)
				log.Debugf("Network updating topology of node: %s", n.node.id)
				if err := n.node.UpdatePeer(peer); err != nil {
					log.Debugf("Network failed to update peers: %v", err)
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
				log.Debugf("Network received close message from: %s", pbNetClose.Node.Id)
				peer := &node{
					id:      pbNetClose.Node.Id,
					address: pbNetClose.Node.Address,
				}
				if err := n.DeletePeerNode(peer.id); err != nil {
					log.Debugf("Network failed to delete node %s routes: %v", peer.id, err)
				}
				if err := n.prunePeerRoutes(peer); err != nil {
					log.Debugf("Network failed pruning peer %s routes: %v", peer.id, err)
				}
			}
		case <-n.closed:
			return
		}
	}
}

// sendMsg sends a message to the tunnel channel
func (n *network) sendMsg(method string, msg proto.Message, channel string) error {
	body, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	// create transport message and chuck it down the pipe
	m := transport.Message{
		Header: map[string]string{
			"Micro-Method": method,
		},
		Body: body,
	}

	// check if the channel client is initialized
	n.RLock()
	client, ok := n.tunClient[channel]
	if !ok || client == nil {
		n.RUnlock()
		return ErrClientNotFound
	}
	n.RUnlock()

	log.Debugf("Network sending %s message from: %s", method, n.options.Id)
	if err := client.Send(&m); err != nil {
		return err
	}

	return nil
}

// announce announces node peers to the network
func (n *network) announce(client transport.Client) {
	announce := time.NewTicker(AnnounceTime)
	defer announce.Stop()

	for {
		select {
		case <-n.closed:
			return
		case <-announce.C:
			msg := PeersToProto(n.node, MaxDepth)
			// advertise yourself to the network
			if err := n.sendMsg("peer", msg, NetworkChannel); err != nil {
				log.Debugf("Network failed to advertise peers: %v", err)
				continue
			}
		}
	}
}

// pruneRoutes prunes routes return by given query
func (n *network) pruneRoutes(q router.Query) error {
	routes, err := n.router.Table().Query(q)
	if err != nil && err != router.ErrRouteNotFound {
		return err
	}

	for _, route := range routes {
		if err := n.router.Table().Delete(route); err != nil && err != router.ErrRouteNotFound {
			return err
		}
	}

	return nil
}

// pruneNodeRoutes prunes routes that were either originated by or routable via given node
func (n *network) prunePeerRoutes(peer *node) error {
	// lookup all routes originated by router
	q := router.NewQuery(
		router.QueryRouter(peer.id),
	)
	if err := n.pruneRoutes(q); err != nil {
		return err
	}

	// lookup all routes routable via gw
	q = router.NewQuery(
		router.QueryGateway(peer.id),
	)
	if err := n.pruneRoutes(q); err != nil {
		return err
	}

	return nil
}

// prune deltes node peers that have not been seen for longer than PruneTime seconds
// prune also removes all the routes either originated by or routable by the stale nodes
func (n *network) prune() {
	prune := time.NewTicker(PruneTime)
	defer prune.Stop()

	for {
		select {
		case <-n.closed:
			return
		case <-prune.C:
			pruned := n.PruneStalePeerNodes(PruneTime)
			for id, peer := range pruned {
				log.Debugf("Network peer exceeded prune time: %s", id)
				if err := n.prunePeerRoutes(peer); err != nil {
					log.Debugf("Network failed pruning peer %s routes: %v", id, err)
				}
			}
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
// - Routes with ID of adjacent nodes are 10
// - Routes by peers of the advertiser are 100
// - Routes beyond your neighbourhood are 1000
func (n *network) setRouteMetric(route *router.Route) {
	// we are the origin of the route
	if route.Router == n.options.Id {
		route.Metric = 1
		return
	}

	// check if the route origin is our peer
	if _, ok := n.peers[route.Router]; ok {
		route.Metric = 10
		return
	}

	// check if the route origin is the peer of our peer
	for _, peer := range n.peers {
		for id := range peer.peers {
			if route.Router == id {
				route.Metric = 100
				return
			}
		}
	}

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
				pbRtrAdvert := &pbRtr.Advert{}
				if err := proto.Unmarshal(m.Body, pbRtrAdvert); err != nil {
					log.Debugf("Network fail to unmarshal advert message: %v", err)
					continue
				}
				// don't process your own messages
				if pbRtrAdvert.Id == n.options.Id {
					continue
				}
				log.Debugf("Network received advert message from: %s", pbRtrAdvert.Id)
				// loookup advertising node in our peer topology
				advertNode := n.node.GetPeerNode(pbRtrAdvert.Id)
				if advertNode == nil {
					// if we can't find the node in our topology (MaxDepth) we skipp prcessing adverts
					log.Debugf("Network skipping advert message from unknown peer: %s", pbRtrAdvert.Id)
					continue
				}

				var events []*router.Event
				for _, event := range pbRtrAdvert.Events {
					// we know the advertising node is not the origin of the route
					if pbRtrAdvert.Id != event.Route.Router {
						// if the origin router is not the advertising node peer
						// we can't rule out potential routing loops so we bail here
						if peer := advertNode.GetPeerNode(event.Route.Router); peer == nil {
							log.Debugf("Network skipping advert message from peer: %s", pbRtrAdvert.Id)
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
					n.node.RLock()
					n.setRouteMetric(&route)
					n.node.RUnlock()
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
				// if no events are eligible for processing continue
				if len(events) == 0 {
					log.Debugf("Network no events to be processed by router: %s", n.options.Id)
					continue
				}
				// create an advert and process it
				advert := &router.Advert{
					Id:        pbRtrAdvert.Id,
					Type:      router.AdvertType(pbRtrAdvert.Type),
					Timestamp: time.Unix(0, pbRtrAdvert.Timestamp),
					TTL:       time.Duration(pbRtrAdvert.Ttl),
					Events:    events,
				}

				log.Debugf("Network router %s processing advert: %s", n.Id(), advert.Id)
				if err := n.router.Process(advert); err != nil {
					log.Debugf("Network failed to process advert %s: %v", advert.Id, err)
				}
			case "solicit":
				pbRtrSolicit := &pbRtr.Solicit{}
				if err := proto.Unmarshal(m.Body, pbRtrSolicit); err != nil {
					log.Debugf("Network fail to unmarshal solicit message: %v", err)
					continue
				}
				log.Debugf("Network received solicit message from: %s", pbRtrSolicit.Id)
				// ignore solicitation when requested by you
				if pbRtrSolicit.Id == n.options.Id {
					continue
				}
				log.Debugf("Network router flushing routes for: %s", pbRtrSolicit.Id)
				// advertise all the routes when a new node has connected
				if err := n.router.Solicit(); err != nil {
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
	hasher := fnv.New64()
	for {
		select {
		// process local adverts and randomly fire them at other nodes
		case advert := <-advertChan:
			// create a proto advert
			var events []*pbRtr.Event
			for _, event := range advert.Events {
				// the routes service address
				address := event.Route.Address

				// only hash the address if we're advertising our own local routes
				if event.Route.Router == advert.Id {
					// hash the service before advertising it
					hasher.Reset()
					hasher.Write([]byte(event.Route.Address + n.node.id))
					address = fmt.Sprintf("%d", hasher.Sum64())
				}

				// NOTE: we override Gateway, Link and Address here
				// TODO: should we avoid overriding gateway?
				route := &pbRtr.Route{
					Service: event.Route.Service,
					Address: address,
					Gateway: n.node.id,
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
			msg := &pbRtr.Advert{
				Id:        advert.Id,
				Type:      pbRtr.AdvertType(advert.Type),
				Timestamp: advert.Timestamp.UnixNano(),
				Events:    events,
			}
			if err := n.sendMsg("advert", msg, ControlChannel); err != nil {
				log.Debugf("Network failed to advertise routes: %v", err)
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
		n.Unlock()
		return nil
	}

	// try to resolve network nodes
	nodes, err := n.resolveNodes()
	if err != nil {
		log.Debugf("Network failed to resolve nodes: %v", err)
	}

	// connect network tunnel
	if err := n.tunnel.Connect(); err != nil {
		n.Unlock()
		return err
	}

	// set our internal node address
	// if advertise address is not set
	if len(n.options.Advertise) == 0 {
		n.node.address = n.tunnel.Address()
		n.server.Init(server.Advertise(n.tunnel.Address()))
	}

	// initialize the tunnel to resolved nodes
	n.tunnel.Init(
		tunnel.Nodes(nodes...),
	)

	// dial into ControlChannel to send route adverts
	ctrlClient, err := n.tunnel.Dial(ControlChannel, tunnel.DialMulticast())
	if err != nil {
		n.Unlock()
		return err
	}

	n.tunClient[ControlChannel] = ctrlClient

	// listen on ControlChannel
	ctrlListener, err := n.tunnel.Listen(ControlChannel)
	if err != nil {
		n.Unlock()
		return err
	}

	// dial into NetworkChannel to send network messages
	netClient, err := n.tunnel.Dial(NetworkChannel, tunnel.DialMulticast())
	if err != nil {
		n.Unlock()
		return err
	}

	n.tunClient[NetworkChannel] = netClient

	// listen on NetworkChannel
	netListener, err := n.tunnel.Listen(NetworkChannel)
	if err != nil {
		n.Unlock()
		return err
	}

	// create closed channel
	n.closed = make(chan bool)

	// start the router
	if err := n.options.Router.Start(); err != nil {
		n.Unlock()
		return err
	}

	// start advertising routes
	advertChan, err := n.options.Router.Advertise()
	if err != nil {
		n.Unlock()
		return err
	}

	// start the server
	if err := n.server.Start(); err != nil {
		n.Unlock()
		return err
	}
	n.Unlock()

	// send connect message to NetworkChannel
	// NOTE: in theory we could do this as soon as
	// Dial to NetworkChannel succeeds, but instead
	// we initialize all other node resources first
	msg := &pbNet.Connect{
		Node: &pbNet.Node{
			Id:      n.node.id,
			Address: n.node.address,
		},
	}
	if err := n.sendMsg("connect", msg, NetworkChannel); err != nil {
		log.Debugf("Network failed to send connect message: %s", err)
	}

	// go resolving network nodes
	go n.resolve()
	// broadcast peers
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

func (n *network) close() error {
	// stop the server
	if err := n.server.Stop(); err != nil {
		return err
	}

	// stop the router
	if err := n.router.Stop(); err != nil {
		return err
	}

	// close the tunnel
	if err := n.tunnel.Close(); err != nil {
		return err
	}

	return nil
}

// Close closes network connection
func (n *network) Close() error {
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

		msg := &pbNet.Close{
			Node: &pbNet.Node{
				Id:      n.node.id,
				Address: n.node.address,
			},
		}
		if err := n.sendMsg("close", msg, NetworkChannel); err != nil {
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
