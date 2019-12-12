package network

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/client"
	rtr "github.com/micro/go-micro/client/selector/router"
	"github.com/micro/go-micro/network/resolver/dns"
	pbNet "github.com/micro/go-micro/network/service/proto"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/router"
	pbRtr "github.com/micro/go-micro/router/service/proto"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/tunnel"
	bun "github.com/micro/go-micro/tunnel/broker"
	tun "github.com/micro/go-micro/tunnel/transport"
	"github.com/micro/go-micro/util/backoff"
	"github.com/micro/go-micro/util/log"
)

var (
	// NetworkChannel is the name of the tunnel channel for passing network messages
	NetworkChannel = "network"
	// ControlChannel is the name of the tunnel channel for passing control message
	ControlChannel = "control"
	// DefaultLink is default network link
	DefaultLink = "network"
	// MaxConnections is the max number of network client connections
	MaxConnections = 3
)

var (
	// ErrClientNotFound is returned when client for tunnel channel could not be found
	ErrClientNotFound = errors.New("client not found")
	// ErrPeerLinkNotFound is returned when peer link could not be found in tunnel Links
	ErrPeerLinkNotFound = errors.New("peer link not found")
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
	// peerLinks is a map of links for each peer
	peerLinks map[string]tunnel.Link

	sync.RWMutex
	// connected marks the network as connected
	connected bool
	// closed closes the network
	closed chan bool
	// whether we've discovered by the network
	discovered chan bool
	// solicted checks whether routes were solicited by one node
	solicited chan *node
}

// message is network message
type message struct {
	// msg is transport message
	msg *transport.Message
	// session is tunnel session
	session tunnel.Session
}

// newNetwork returns a new network node
func newNetwork(opts ...Option) Network {
	rand.Seed(time.Now().UnixNano())
	options := DefaultOptions()

	for _, o := range opts {
		o(&options)
	}

	// set the address to a hashed address
	hasher := fnv.New64()
	hasher.Write([]byte(options.Address + options.Id))
	address := fmt.Sprintf("%d", hasher.Sum64())

	// set the address to advertise
	var advertise string
	var peerAddress string

	if len(options.Advertise) > 0 {
		advertise = options.Advertise
		peerAddress = options.Advertise
	} else {
		advertise = options.Address
		peerAddress = address
	}

	// init tunnel address to the network bind address
	options.Tunnel.Init(
		tunnel.Address(options.Address),
	)

	// init router Id to the network id
	options.Router.Init(
		router.Id(options.Id),
		router.Address(peerAddress),
	)

	// create tunnel client with tunnel transport
	tunTransport := tun.NewTransport(
		tun.WithTunnel(options.Tunnel),
	)

	// create the tunnel broker
	tunBroker := bun.NewBroker(
		bun.WithTunnel(options.Tunnel),
	)

	// server is network server
	server := server.NewServer(
		server.Id(options.Id),
		server.Address(peerAddress),
		server.Advertise(advertise),
		server.Name(options.Name),
		server.Transport(tunTransport),
		server.Broker(tunBroker),
	)

	// client is network client
	client := client.NewClient(
		client.Broker(tunBroker),
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
			address: peerAddress,
			peers:   make(map[string]*node),
		},
		options:    options,
		router:     options.Router,
		proxy:      options.Proxy,
		tunnel:     options.Tunnel,
		server:     server,
		client:     client,
		tunClient:  make(map[string]transport.Client),
		peerLinks:  make(map[string]tunnel.Link),
		discovered: make(chan bool, 1),
		solicited:  make(chan *node, 1),
	}

	network.node.network = network

	return network
}

func (n *network) Init(opts ...Option) error {
	n.Lock()
	// TODO: maybe only allow reinit of certain opts
	for _, o := range opts {
		o(&n.options)
	}
	n.Unlock()

	return nil
}

// Options returns network options
func (n *network) Options() Options {
	n.RLock()
	options := n.options
	n.RUnlock()
	return options
}

// Name returns network name
func (n *network) Name() string {
	return n.options.Name
}

// acceptNetConn accepts connections from NetworkChannel
func (n *network) acceptNetConn(l tunnel.Listener, recv chan *message) {
	var i int
	for {
		// accept a connection
		conn, err := l.Accept()
		if err != nil {
			sleep := backoff.Do(i)
			log.Debugf("Network tunnel [%s] accept error: %v, backing off for %v", ControlChannel, err, sleep)
			time.Sleep(sleep)
			if i > 5 {
				i = 0
			}
			i++
			continue
		}

		select {
		case <-n.closed:
			if err := conn.Close(); err != nil {
				log.Debugf("Network tunnel [%s] failed to close connection: %v", NetworkChannel, err)
			}
			return
		default:
			// go handle NetworkChannel connection
			go n.handleNetConn(conn, recv)
		}
	}
}

// acceptCtrlConn accepts connections from ControlChannel
func (n *network) acceptCtrlConn(l tunnel.Listener, recv chan *message) {
	var i int
	for {
		// accept a connection
		conn, err := l.Accept()
		if err != nil {
			sleep := backoff.Do(i)
			log.Debugf("Network tunnel [%s] accept error: %v, backing off for %v", ControlChannel, err, sleep)
			time.Sleep(sleep)
			if i > 5 {
				// reset the counter
				i = 0
			}
			i++
			continue
		}

		select {
		case <-n.closed:
			if err := conn.Close(); err != nil {
				log.Debugf("Network tunnel [%s] failed to close connection: %v", ControlChannel, err)
			}
			return
		default:
			// go handle ControlChannel connection
			go n.handleCtrlConn(conn, recv)
		}
	}
}

// handleCtrlConn handles ControlChannel connections
// advertise advertises routes to the network
func (n *network) advertise(advertChan <-chan *router.Advert) {
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
					// routes for multiple instances of a service will be collapsed here.
					// TODO: once we store labels in the table this may need to change
					// to include the labels in case they differ but highly unlikely
					hasher.Write([]byte(event.Route.Service + n.node.Address()))
					address = fmt.Sprintf("%d", hasher.Sum64())
				}
				// calculate route metric to advertise
				metric := n.getRouteMetric(event.Route.Router, event.Route.Gateway, event.Route.Link)
				// NOTE: we override Gateway, Link and Address here
				route := &pbRtr.Route{
					Service: event.Route.Service,
					Address: address,
					Gateway: n.node.Address(),
					Network: event.Route.Network,
					Router:  event.Route.Router,
					Link:    DefaultLink,
					Metric:  metric,
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

			// send the advert to all on the control channel
			// since its not a solicitation
			if advert.Type != router.Solicitation {
				if err := n.sendMsg("advert", ControlChannel, msg); err != nil {
					log.Debugf("Network failed to advertise routes: %v", err)
				}
				continue
			}

			// it's a solication, someone asked for it
			// so we're going to pick off the node and send it
			select {
			case peer := <-n.solicited:
				// someone requested the route
				n.sendTo("advert", ControlChannel, peer, msg)
			default:
				if err := n.sendMsg("advert", ControlChannel, msg); err != nil {
					log.Debugf("Network failed to advertise routes: %v", err)
				}
			}
		case <-n.closed:
			return
		}
	}
}

func (n *network) initNodes(startup bool) {
	nodes, err := n.resolveNodes()
	if err != nil && !startup {
		log.Debugf("Network failed to resolve nodes: %v", err)
		return
	}

	// initialize the tunnel
	log.Tracef("Network initialising nodes %+v\n", nodes)

	n.tunnel.Init(
		tunnel.Nodes(nodes...),
	)
}

// resolveNodes resolves network nodes to addresses
func (n *network) resolveNodes() ([]string, error) {
	// resolve the network address to network nodes
	records, err := n.options.Resolver.Resolve(n.options.Name)
	if err != nil {
		log.Debugf("Network failed to resolve nodes: %v", err)
	}

	// sort by lowest priority
	if err == nil {
		sort.Slice(records, func(i, j int) bool { return records[i].Priority < records[j].Priority })
	}

	// keep processing

	nodeMap := make(map[string]bool)

	// collect network node addresses
	//nolint:prealloc
	var nodes []string
	var i int

	for _, record := range records {
		if _, ok := nodeMap[record.Address]; ok {
			continue
		}

		nodeMap[record.Address] = true
		nodes = append(nodes, record.Address)

		i++

		// break once MaxConnection nodes has been reached
		if i == MaxConnections {
			break
		}
	}

	// use the dns resolver to expand peers
	dns := &dns.Resolver{}

	// append seed nodes if we have them
	for _, node := range n.options.Nodes {
		// resolve anything that looks like a host name
		records, err := dns.Resolve(node)
		if err != nil {
			log.Debugf("Failed to resolve %v %v", node, err)
			continue
		}

		// add to the node map
		for _, record := range records {
			if _, ok := nodeMap[record.Address]; !ok {
				nodes = append(nodes, record.Address)
			}
		}
	}

	return nodes, nil
}

// handleNetConn handles network announcement messages
func (n *network) handleNetConn(s tunnel.Session, msg chan *message) {
	for {
		m := new(transport.Message)
		if err := s.Recv(m); err != nil {
			log.Debugf("Network tunnel [%s] receive error: %v", NetworkChannel, err)
			switch err {
			case io.EOF, tunnel.ErrReadTimeout:
				s.Close()
				return
			}
			continue
		}

		// check if peer is set
		peer := m.Header["Micro-Peer"]

		// check who the message is intended for
		if len(peer) > 0 && peer != n.options.Id {
			continue
		}

		select {
		case msg <- &message{
			msg:     m,
			session: s,
		}:
		case <-n.closed:
			return
		}
	}
}

func (n *network) handleCtrlConn(s tunnel.Session, msg chan *message) {
	for {
		m := new(transport.Message)
		if err := s.Recv(m); err != nil {
			log.Debugf("Network tunnel [%s] receive error: %v", ControlChannel, err)
			switch err {
			case io.EOF, tunnel.ErrReadTimeout:
				s.Close()
				return
			}
			continue
		}

		// check if peer is set
		peer := m.Header["Micro-Peer"]

		// check who the message is intended for
		if len(peer) > 0 && peer != n.options.Id {
			continue
		}

		select {
		case msg <- &message{
			msg:     m,
			session: s,
		}:
		case <-n.closed:
			return
		}
	}
}

// getHopCount queries network graph and returns hop count for given router
// - Routes for local services have hop count 1
// - Routes with ID of adjacent nodes have hop count 2
// - Routes by peers of the advertiser have hop count 3
// - Routes beyond node neighbourhood have hop count 4
func (n *network) getHopCount(rtr string) int {
	// make sure node.peers are not modified
	n.node.RLock()
	defer n.node.RUnlock()

	// we are the origin of the route
	if rtr == n.options.Id {
		return 1
	}

	// the route origin is our peer
	if _, ok := n.node.peers[rtr]; ok {
		return 10
	}

	// the route origin is the peer of our peer
	for _, peer := range n.node.peers {
		for id := range peer.peers {
			if rtr == id {
				return 100
			}
		}
	}
	// otherwise we are three hops away
	return 1000
}

// getRouteMetric calculates router metric and returns it
// Route metric is calculated based on link status and route hopd count
func (n *network) getRouteMetric(router string, gateway string, link string) int64 {
	// set the route metric
	n.RLock()
	defer n.RUnlock()

	// local links are marked as 1
	if link == "local" && gateway == "" {
		return 1
	}

	// local links from other gateways as 2
	if link == "local" && gateway != "" {
		return 2
	}

	log.Tracef("Network looking up %s link to gateway: %s", link, gateway)

	// attempt to find link based on gateway address
	lnk, ok := n.peerLinks[gateway]
	if !ok {
		log.Debugf("Network failed to find a link to gateway: %s", gateway)
		// no link found so infinite metric returned
		return math.MaxInt64
	}

	// calculating metric

	delay := lnk.Delay()
	hops := n.getHopCount(router)
	length := lnk.Length()

	// make sure delay is non-zero
	if delay == 0 {
		delay = 1
	}

	// make sure length is non-zero
	if length == 0 {
		log.Debugf("Link length is 0 %v %v", link, lnk.Length())
		length = 10e9
	}

	log.Tracef("Network calculated metric %v delay %v length %v distance %v", (delay*length*int64(hops))/10e6, delay, length, hops)

	return (delay * length * int64(hops)) / 10e6
}

// processCtrlChan processes messages received on ControlChannel
func (n *network) processCtrlChan(listener tunnel.Listener) {
	defer listener.Close()

	// receive control message queue
	recv := make(chan *message, 128)

	// accept ControlChannel cconnections
	go n.acceptCtrlConn(listener, recv)

	for {
		select {
		case m := <-recv:
			// switch on type of message and take action
			switch m.msg.Header["Micro-Method"] {
			case "advert":
				pbRtrAdvert := &pbRtr.Advert{}

				if err := proto.Unmarshal(m.msg.Body, pbRtrAdvert); err != nil {
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
						Metric:  event.Route.Metric,
					}

					// calculate route metric and add to the advertised metric
					// we need to make sure we do not overflow math.MaxInt64
					metric := n.getRouteMetric(event.Route.Router, event.Route.Gateway, event.Route.Link)
					log.Tracef("Network metric for router %s and gateway %s: %v", event.Route.Router, event.Route.Gateway, metric)

					// check we don't overflow max int 64
					if d := route.Metric + metric; d <= 0 {
						// set to max int64 if we overflow
						route.Metric = math.MaxInt64
					} else {
						// set the combined value of metrics otherwise
						route.Metric = d
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
					log.Tracef("Network no events to be processed by router: %s", n.options.Id)
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

				log.Tracef("Network router %s processing advert: %s", n.Id(), advert.Id)
				if err := n.router.Process(advert); err != nil {
					log.Debugf("Network failed to process advert %s: %v", advert.Id, err)
				}
			case "solicit":
				pbRtrSolicit := &pbRtr.Solicit{}
				if err := proto.Unmarshal(m.msg.Body, pbRtrSolicit); err != nil {
					log.Debugf("Network fail to unmarshal solicit message: %v", err)
					continue
				}

				log.Debugf("Network received solicit message from: %s", pbRtrSolicit.Id)

				// ignore solicitation when requested by you
				if pbRtrSolicit.Id == n.options.Id {
					continue
				}

				log.Tracef("Network router flushing routes for: %s", pbRtrSolicit.Id)

				peer := &node{
					id:   pbRtrSolicit.Id,
					link: m.msg.Header["Micro-Link"],
				}

				// specify that someone solicited the route
				select {
				case n.solicited <- peer:
				default:
					// don't block
				}

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

// processNetChan processes messages received on NetworkChannel
func (n *network) processNetChan(listener tunnel.Listener) {
	defer listener.Close()

	// receive network message queue
	recv := make(chan *message, 128)

	// accept NetworkChannel connections
	go n.acceptNetConn(listener, recv)

	for {
		select {
		case m := <-recv:
			// switch on type of message and take action
			switch m.msg.Header["Micro-Method"] {
			case "connect":
				// mark the time the message has been received
				now := time.Now()
				pbNetConnect := &pbNet.Connect{}

				if err := proto.Unmarshal(m.msg.Body, pbNetConnect); err != nil {
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
					link:     m.msg.Header["Micro-Link"],
					peers:    make(map[string]*node),
					lastSeen: now,
				}

				// update peer links

				if err := n.updatePeerLinks(peer); err != nil {
					log.Debugf("Network failed updating peer links: %s", err)
				}

				// add peer to the list of node peers
				if err := n.node.AddPeer(peer); err == ErrPeerExists {
					log.Tracef("Network peer exists, refreshing: %s", peer.id)
					// update lastSeen time for the existing node
					if err := n.RefreshPeer(peer.id, peer.link, now); err != nil {
						log.Debugf("Network failed refreshing peer %s: %v", peer.id, err)
					}
				}

				// we send the peer message because someone has sent connect
				// and wants to know what's on the network. The faster we
				// respond the faster we start to converge

				// get node peers down to MaxDepth encoded in protobuf
				msg := PeersToProto(n.node, MaxDepth)

				go func() {
					// advertise yourself to the network
					if err := n.sendTo("peer", NetworkChannel, peer, msg); err != nil {
						log.Debugf("Network failed to advertise peers: %v", err)
					}

					<-time.After(time.Millisecond * 100)

					// specify that we're soliciting
					select {
					case n.solicited <- peer:
					default:
						// don't block
					}

					// advertise all the routes when a new node has connected
					if err := n.router.Solicit(); err != nil {
						log.Debugf("Network failed to solicit routes: %s", err)
					}

				}()
			case "peer":
				// mark the time the message has been received
				now := time.Now()
				pbNetPeer := &pbNet.Peer{}

				if err := proto.Unmarshal(m.msg.Body, pbNetPeer); err != nil {
					log.Debugf("Network tunnel [%s] peer unmarshal error: %v", NetworkChannel, err)
					continue
				}

				// don't process your own messages
				if pbNetPeer.Node.Id == n.options.Id {
					continue
				}

				log.Debugf("Network received peer message from: %s %s", pbNetPeer.Node.Id, pbNetPeer.Node.Address)

				peer := &node{
					id:       pbNetPeer.Node.Id,
					address:  pbNetPeer.Node.Address,
					link:     m.msg.Header["Micro-Link"],
					peers:    make(map[string]*node),
					lastSeen: now,
				}

				// update peer links

				if err := n.updatePeerLinks(peer); err != nil {
					log.Debugf("Network failed updating peer links: %s", err)
				}

				if err := n.node.AddPeer(peer); err == nil {
					// send a solicit message when discovering new peer
					msg := &pbRtr.Solicit{
						Id: n.options.Id,
					}

					go func() {
						// advertise yourself to the peer
						if err := n.sendTo("peer", NetworkChannel, peer, msg); err != nil {
							log.Debugf("Network failed to advertise peers: %v", err)
						}

						// wait for a second
						<-time.After(time.Millisecond * 100)

						// then solicit this peer
						if err := n.sendTo("solicit", ControlChannel, peer, msg); err != nil {
							log.Debugf("Network failed to send solicit message: %s", err)
						}
					}()

					continue
					// we're expecting any error to be ErrPeerExists
				} else if err != ErrPeerExists {
					log.Debugf("Network got error adding peer %v", err)
					continue
				}

				log.Tracef("Network peer exists, refreshing: %s", pbNetPeer.Node.Id)

				// update lastSeen time for the peer
				if err := n.RefreshPeer(pbNetPeer.Node.Id, peer.link, now); err != nil {
					log.Debugf("Network failed refreshing peer %s: %v", pbNetPeer.Node.Id, err)
				}

				// NOTE: we don't unpack MaxDepth toplogy
				peer = UnpackPeerTopology(pbNetPeer, now, MaxDepth-1)
				log.Tracef("Network updating topology of node: %s", n.node.id)
				if err := n.node.UpdatePeer(peer); err != nil {
					log.Debugf("Network failed to update peers: %v", err)
				}

				// tell the connect loop that we've been discovered
				// so it stops sending connect messages out
				select {
				case n.discovered <- true:
				default:
					// don't block here
				}
			case "close":
				pbNetClose := &pbNet.Close{}
				if err := proto.Unmarshal(m.msg.Body, pbNetClose); err != nil {
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

				// delete peer from the peerLinks
				n.Lock()
				delete(n.peerLinks, pbNetClose.Node.Address)
				n.Unlock()
			}
		case <-n.closed:
			return
		}
	}
}

// pruneRoutes prunes routes return by given query
func (n *network) pruneRoutes(q ...router.QueryOption) error {
	routes, err := n.router.Table().Query(q...)
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
	q := []router.QueryOption{
		router.QueryRouter(peer.id),
	}
	if err := n.pruneRoutes(q...); err != nil {
		return err
	}

	// lookup all routes routable via gw
	q = []router.QueryOption{
		router.QueryGateway(peer.address),
	}
	if err := n.pruneRoutes(q...); err != nil {
		return err
	}

	return nil
}

// manage the process of announcing to peers and prune any peer nodes that have not been
// seen for a period of time. Also removes all the routes either originated by or routable
//by the stale nodes. it also resolves nodes periodically and adds them to the tunnel
func (n *network) manage() {
	announce := time.NewTicker(AnnounceTime)
	defer announce.Stop()
	prune := time.NewTicker(PruneTime)
	defer prune.Stop()
	resolve := time.NewTicker(ResolveTime)
	defer resolve.Stop()

	for {
		select {
		case <-n.closed:
			return
		case <-announce.C:
			// jitter
			j := rand.Int63n(int64(AnnounceTime.Seconds() / 2.0))
			time.Sleep(time.Duration(j) * time.Second)

			// TODO: intermittently flip between peer selection
			// and full broadcast pick a random set of peers

			msg := PeersToProto(n.node, MaxDepth)
			// advertise yourself to the network
			if err := n.sendMsg("peer", NetworkChannel, msg); err != nil {
				log.Debugf("Network failed to advertise peers: %v", err)
			}
		case <-prune.C:
			pruned := n.PruneStalePeers(PruneTime)

			for id, peer := range pruned {
				log.Debugf("Network peer exceeded prune time: %s", id)

				n.Lock()
				delete(n.peerLinks, peer.address)
				n.Unlock()

				if err := n.prunePeerRoutes(peer); err != nil {
					log.Debugf("Network failed pruning peer %s routes: %v", id, err)
				}
			}

			// get a list of all routes
			routes, err := n.options.Router.Table().List()
			if err != nil {
				log.Debugf("Network failed listing routes when pruning peers: %v", err)
				continue
			}

			// collect all the router IDs in the routing table
			routers := make(map[string]bool)

			for _, route := range routes {
				// check if its been processed
				if _, ok := routers[route.Router]; ok {
					continue
				}

				// mark as processed
				routers[route.Router] = true

				// if the router is NOT in our peer graph, delete all routes originated by it
				if peer := n.node.GetPeerNode(route.Router); peer != nil {
					continue
				}

				if err := n.pruneRoutes(router.QueryRouter(route.Router)); err != nil {
					log.Debugf("Network failed deleting routes by %s: %v", route.Router, err)
				}
			}
		case <-resolve.C:
			n.initNodes(false)
		}
	}
}

func (n *network) sendConnect() {
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

	if err := n.sendMsg("connect", NetworkChannel, msg); err != nil {
		log.Debugf("Network failed to send connect message: %s", err)
	}
}

// sendTo sends a message to a specific node as a one off.
// we need this because when links die, we have no discovery info,
// and sending to an existing multicast link doesn't immediately work
func (n *network) sendTo(method, channel string, peer *node, msg proto.Message) error {
	body, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	// Create a unicast connection to the peer but don't do the open/accept flow
	c, err := n.tunnel.Dial(channel, tunnel.DialWait(false), tunnel.DialLink(peer.link))
	if err != nil {
		return err
	}
	defer c.Close()

	log.Debugf("Network sending %s message from: %s to %s", method, n.options.Id, peer.id)

	return c.Send(&transport.Message{
		Header: map[string]string{
			"Micro-Method": method,
			"Micro-Peer":   peer.id,
		},
		Body: body,
	})
}

// sendMsg sends a message to the tunnel channel
func (n *network) sendMsg(method, channel string, msg proto.Message) error {
	body, err := proto.Marshal(msg)
	if err != nil {
		return err
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

	return client.Send(&transport.Message{
		Header: map[string]string{
			"Micro-Method": method,
		},
		Body: body,
	})
}

// updatePeerLinks updates link for a given peer
func (n *network) updatePeerLinks(peer *node) error {
	n.Lock()
	defer n.Unlock()

	linkId := peer.link

	log.Tracef("Network looking up link %s in the peer links", linkId)

	// lookup the peer link
	var peerLink tunnel.Link

	for _, link := range n.tunnel.Links() {
		if link.Id() == linkId {
			peerLink = link
			break
		}
	}

	if peerLink == nil {
		return ErrPeerLinkNotFound
	}

	// if the peerLink is found in the returned links update peerLinks
	log.Tracef("Network updating peer links for peer %s", peer.address)

	// add peerLink to the peerLinks map
	if link, ok := n.peerLinks[peer.address]; ok {
		// if the existing has better Length then the new, replace it
		if link.Length() < peerLink.Length() {
			n.peerLinks[peer.address] = peerLink
		}
	} else {
		n.peerLinks[peer.address] = peerLink
	}

	return nil
}

// connect will wait for a link to be established and send the connect
// message. We're trying to ensure convergence pretty quickly. So we want
// to hear back. In the case we become completely disconnected we'll
// connect again once a new link is established
func (n *network) connect() {
	// discovered lets us know what we received a peer message back
	var discovered bool
	var attempts int

	// our advertise address
	loopback := n.server.Options().Advertise
	// actual address
	address := n.tunnel.Address()

	for {
		// connected is used to define if the link is connected
		var connected bool

		// check the links state
		for _, link := range n.tunnel.Links() {
			// skip loopback
			if link.Loopback() {
				continue
			}

			// if remote is ourselves
			switch link.Remote() {
			case loopback, address:
				continue
			}

			if link.State() == "connected" {
				connected = true
				break
			}
		}

		// if we're not connected wait
		if !connected {
			// reset discovered
			discovered = false
			// sleep for a second
			time.Sleep(time.Second)
			// now try again
			continue
		}

		// we're connected but are we discovered?
		if !discovered {
			// recreate the clients because all the tunnel links are gone
			// so we haven't send discovery beneath
			if err := n.createClients(); err != nil {
				log.Debugf("Failed to recreate network/control clients: %v", err)
				continue
			}

			// send the connect message
			n.sendConnect()
		}

		// check if we've been discovered
		select {
		case <-n.discovered:
			discovered = true
			attempts = 0
		case <-n.closed:
			return
		case <-time.After(time.Second + backoff.Do(attempts)):
			// we have to try again
			attempts++

			// reset attempts 5 == ~2mins
			if attempts > 5 {
				attempts = 0
			}
		}
	}
}

// Connect connects the network
func (n *network) Connect() error {
	n.Lock()
	defer n.Unlock()

	// connect network tunnel
	if err := n.tunnel.Connect(); err != nil {
		return err
	}

	// return if already connected
	if n.connected {
		// initialise the nodes
		n.initNodes(false)
		// send the connect message
		go n.sendConnect()
		return nil
	}

	// initialise the nodes
	n.initNodes(true)

	// set our internal node address
	// if advertise address is not set
	if len(n.options.Advertise) == 0 {
		n.server.Init(server.Advertise(n.tunnel.Address()))
	}

	// listen on NetworkChannel
	netListener, err := n.tunnel.Listen(
		NetworkChannel,
		tunnel.ListenMode(tunnel.Multicast),
		tunnel.ListenTimeout(AnnounceTime*2),
	)
	if err != nil {
		return err
	}

	// listen on ControlChannel
	ctrlListener, err := n.tunnel.Listen(
		ControlChannel,
		tunnel.ListenMode(tunnel.Multicast),
		tunnel.ListenTimeout(router.AdvertiseTableTick*2),
	)
	if err != nil {
		return err
	}

	// dial into ControlChannel to send route adverts
	ctrlClient, err := n.tunnel.Dial(ControlChannel, tunnel.DialMode(tunnel.Multicast))
	if err != nil {
		return err
	}

	n.tunClient[ControlChannel] = ctrlClient

	// dial into NetworkChannel to send network messages
	netClient, err := n.tunnel.Dial(NetworkChannel, tunnel.DialMode(tunnel.Multicast))
	if err != nil {
		return err
	}

	n.tunClient[NetworkChannel] = netClient

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

	// advertise service routes
	go n.advertise(advertChan)
	// listen to network messages
	go n.processNetChan(netListener)
	// accept and process routes
	go n.processCtrlChan(ctrlListener)
	// manage connection once links are established
	go n.connect()
	// resolve nodes, broadcast announcements and prune stale nodes
	go n.manage()

	// we're now connected
	n.connected = true

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

// createClients is used to create new clients in the event we lose all the tunnels
func (n *network) createClients() error {
	// dial into ControlChannel to send route adverts
	ctrlClient, err := n.tunnel.Dial(ControlChannel, tunnel.DialMode(tunnel.Multicast))
	if err != nil {
		return err
	}

	// dial into NetworkChannel to send network messages
	netClient, err := n.tunnel.Dial(NetworkChannel, tunnel.DialMode(tunnel.Multicast))
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	// set the control client
	c, ok := n.tunClient[ControlChannel]
	if ok {
		c.Close()
	}
	n.tunClient[ControlChannel] = ctrlClient

	// set the network client
	c, ok = n.tunClient[NetworkChannel]
	if ok {
		c.Close()
	}
	n.tunClient[NetworkChannel] = netClient

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

		if err := n.sendMsg("close", NetworkChannel, msg); err != nil {
			log.Debugf("Network failed to send close message: %s", err)
		}
		<-time.After(time.Millisecond * 100)
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
