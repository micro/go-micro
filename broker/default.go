package broker

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/codec/json"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/util/addr"
	"github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
)

type natsBroker struct {
	sync.Once
	sync.RWMutex

	// indicate if we're connected
	connected bool

	// address to bind routes to
	addrs []string
	// servers for the client
	servers []string

	// client connection and nats opts
	conn  *nats.Conn
	opts  Options
	nopts nats.Options

	// should we drain the connection
	drain   bool
	closeCh chan (error)

	// embedded server
	server *server.Server
	// configure to use local server
	local bool
	// server exit channel
	exit chan bool
}

type subscriber struct {
	s    *nats.Subscription
	opts SubscribeOptions
}

type publication struct {
	t   string
	err error
	m   *Message
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *Message {
	return p.m
}

func (p *publication) Ack() error {
	// nats does not support acking
	return nil
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.s.Subject
}

func (s *subscriber) Unsubscribe() error {
	return s.s.Unsubscribe()
}

func (n *natsBroker) Address() string {
	n.RLock()
	defer n.RUnlock()

	if n.server != nil {
		return n.server.ClusterAddr().String()
	}

	if n.conn != nil && n.conn.IsConnected() {
		return n.conn.ConnectedUrl()
	}

	if len(n.addrs) > 0 {
		return n.addrs[0]
	}

	return "127.0.0.1:-1"
}

func (n *natsBroker) setAddrs(addrs []string) []string {
	//nolint:prealloc
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	// if there's no address and we weren't told to
	// embed a local server then use the default url
	if len(cAddrs) == 0 && !n.local {
		cAddrs = []string{nats.DefaultURL}
	}
	return cAddrs
}

// serve stats a local nats server if needed
func (n *natsBroker) serve(exit chan bool) error {
	// local server address
	host := "127.0.0.1"
	port := -1

	// cluster address
	caddr := "0.0.0.0"
	cport := -1

	// with no address we just default it
	// this is a local client address
	if len(n.addrs) > 0 {
		address := n.addrs[0]
		if strings.HasPrefix(address, "nats://") {
			address = strings.TrimPrefix(address, "nats://")
		}

		// parse out the address
		h, p, err := net.SplitHostPort(address)
		if err == nil {
			caddr = h
			cport, _ = strconv.Atoi(p)
		}
	}

	// 1. create new server
	// 2. register the server
	// 3. connect to other servers

	// set cluster opts
	cOpts := server.ClusterOpts{
		Host: caddr,
		Port: cport,
	}

	// get the routes for other nodes
	var routes []*url.URL

	// get existing nats servers to connect to
	services, err := n.opts.Registry.GetService("go.micro.nats.broker")
	if err == nil {
		for _, service := range services {
			for _, node := range service.Nodes {
				u, err := url.Parse("nats://" + node.Address)
				if err != nil {
					if logger.V(logger.InfoLevel, logger.DefaultLogger) {
						logger.Info(err)
					}
					continue
				}
				// append to the cluster routes
				routes = append(routes, u)
			}
		}
	}

	// try get existing server
	s := n.server

	if s != nil {
		// stop the existing server
		s.Shutdown()
	}

	s, err = server.NewServer(&server.Options{
		// Specify the host
		Host: host,
		// Use a random port
		Port: port,
		// Set the cluster ops
		Cluster: cOpts,
		// Set the routes
		Routes:         routes,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 2048,
		TLSConfig:      n.opts.TLSConfig,
	})
	if err != nil {
		return err
	}

	// save the server
	n.server = s

	// start the server
	go s.Start()

	var ready bool

	// wait till its ready for connections
	for i := 0; i < 3; i++ {
		if s.ReadyForConnections(time.Second) {
			ready = true
			break
		}
	}

	if !ready {
		return errors.New("server not ready")
	}

	// set the client address
	n.servers = []string{s.ClientURL()}

	go func() {
		var advertise string

		// parse out the address
		_, port, err := net.SplitHostPort(s.ClusterAddr().String())
		if err == nil {
			addr, _ := addr.Extract("")
			advertise = net.JoinHostPort(addr, port)
		} else {
			s.ClusterAddr().String()
		}

		// register the cluster address
		for {
			select {
			case err := <-n.closeCh:
				if err != nil {
					if logger.V(logger.InfoLevel, logger.DefaultLogger) {
						logger.Info(err)
					}
				}
			case <-exit:
				// deregister on exit
				n.opts.Registry.Deregister(&registry.Service{
					Name:    "go.micro.nats.broker",
					Version: "v2",
					Nodes: []*registry.Node{
						{Id: s.ID(), Address: advertise},
					},
				})
				s.Shutdown()
				return
			default:
				// register the broker
				n.opts.Registry.Register(&registry.Service{
					Name:    "go.micro.nats.broker",
					Version: "v2",
					Nodes: []*registry.Node{
						{Id: s.ID(), Address: advertise},
					},
				}, registry.RegisterTTL(time.Minute))
				time.Sleep(time.Minute)
			}
		}
	}()

	return nil
}

func (n *natsBroker) Connect() error {
	n.Lock()
	defer n.Unlock()

	if !n.connected {
		// create exit chan
		n.exit = make(chan bool)

		// start the local server
		if err := n.serve(n.exit); err != nil {
			return err
		}

		// set to connected
	}

	status := nats.CLOSED
	if n.conn != nil {
		status = n.conn.Status()
	}

	switch status {
	case nats.CONNECTED, nats.RECONNECTING, nats.CONNECTING:
		return nil
	default: // DISCONNECTED or CLOSED or DRAINING
		opts := n.nopts
		opts.DrainTimeout = 1 * time.Second
		opts.AsyncErrorCB = n.onAsyncError
		opts.DisconnectedErrCB = n.onDisconnectedError
		opts.ClosedCB = n.onClose
		opts.Servers = n.servers
		opts.Secure = n.opts.Secure
		opts.TLSConfig = n.opts.TLSConfig

		// secure might not be set
		if n.opts.TLSConfig != nil {
			opts.Secure = true
		}

		c, err := opts.Connect()
		if err != nil {
			return err
		}
		n.conn = c

		n.connected = true

		return nil
	}
}

func (n *natsBroker) Disconnect() error {
	n.RLock()
	defer n.RUnlock()

	if !n.connected {
		return nil
	}

	// drain the connection if specified
	if n.drain {
		n.conn.Drain()
	}

	// close the client connection
	n.conn.Close()

	// shutdown the local server
	// and deregister
	if n.server != nil {
		select {
		case <-n.exit:
		default:
			close(n.exit)
		}
	}

	// set not connected
	n.connected = false

	return nil
}

func (n *natsBroker) Init(opts ...Option) error {
	n.setOption(opts...)
	return nil
}

func (n *natsBroker) Options() Options {
	return n.opts
}

func (n *natsBroker) Publish(topic string, msg *Message, opts ...PublishOption) error {
	b, err := n.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}
	n.RLock()
	defer n.RUnlock()
	return n.conn.Publish(topic, b)
}

func (n *natsBroker) Subscribe(topic string, handler Handler, opts ...SubscribeOption) (Subscriber, error) {
	if n.conn == nil {
		return nil, errors.New("not connected")
	}

	opt := SubscribeOptions{
		AutoAck: true,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	fn := func(msg *nats.Msg) {
		var m Message
		pub := &publication{t: msg.Subject}
		eh := n.opts.ErrorHandler
		err := n.opts.Codec.Unmarshal(msg.Data, &m)
		pub.err = err
		pub.m = &m
		if err != nil {
			m.Body = msg.Data
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			if eh != nil {
				eh(pub)
			}
			return
		}
		if err := handler(pub); err != nil {
			pub.err = err
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			if eh != nil {
				eh(pub)
			}
		}
	}

	var sub *nats.Subscription
	var err error

	n.RLock()
	if len(opt.Queue) > 0 {
		sub, err = n.conn.QueueSubscribe(topic, opt.Queue, fn)
	} else {
		sub, err = n.conn.Subscribe(topic, fn)
	}
	n.RUnlock()
	if err != nil {
		return nil, err
	}
	return &subscriber{s: sub, opts: opt}, nil
}

func (n *natsBroker) String() string {
	return "eats"
}

func (n *natsBroker) setOption(opts ...Option) {
	for _, o := range opts {
		o(&n.opts)
	}

	n.Once.Do(func() {
		n.nopts = nats.GetDefaultOptions()
	})

	// local embedded server
	n.local = true
	// set to drain
	n.drain = true

	if !n.opts.Secure {
		n.opts.Secure = n.nopts.Secure
	}

	if n.opts.TLSConfig == nil {
		n.opts.TLSConfig = n.nopts.TLSConfig
	}

	n.addrs = n.setAddrs(n.opts.Addrs)
}

func (n *natsBroker) onClose(conn *nats.Conn) {
	n.closeCh <- nil
}

func (n *natsBroker) onDisconnectedError(conn *nats.Conn, err error) {
	n.closeCh <- err
}

func (n *natsBroker) onAsyncError(conn *nats.Conn, sub *nats.Subscription, err error) {
	// There are kinds of different async error nats might callback, but we are interested
	// in ErrDrainTimeout only here.
	if err == nats.ErrDrainTimeout {
		n.closeCh <- err
	}
}

func NewBroker(opts ...Option) Broker {
	options := Options{
		// Default codec
		Codec:    json.Marshaler{},
		Context:  context.Background(),
		Registry: registry.DefaultRegistry,
	}

	n := &natsBroker{
		opts:    options,
		closeCh: make(chan error),
	}
	n.setOption(opts...)

	return n
}
