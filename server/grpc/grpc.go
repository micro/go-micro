// Package grpc provides a grpc server
package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/errors"
	"go-micro.dev/v5/logger"
	meta "go-micro.dev/v5/metadata"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/util/addr"
	"go-micro.dev/v5/util/backoff"
	mgrpc "go-micro.dev/v5/util/grpc"
	mnet "go-micro.dev/v5/util/net"
	"golang.org/x/net/netutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func init() {
	cmd.DefaultServers["grpc"] = NewServer
}

var (
	// DefaultMaxMsgSize define maximum message size that server can send
	// or receive.  Default value is 4MB.
	DefaultMaxMsgSize = 1024 * 1024 * 4
)

const (
	defaultContentType = "application/grpc"
)

type grpcServer struct {
	rpc  *rServer
	srv  *grpc.Server
	exit chan chan error
	wg   *sync.WaitGroup

	sync.RWMutex
	opts        server.Options
	handlers    map[string]server.Handler
	subscribers map[*subscriber][]broker.Subscriber
	// marks the serve as started
	started bool
	// used for first registration
	registered bool

	// registry service instance
	rsvc *registry.Service
}

func init() {
	encoding.RegisterCodec(wrapCodec{jsonCodec{}})
	encoding.RegisterCodec(wrapCodec{protoCodec{}})
	encoding.RegisterCodec(wrapCodec{bytesCodec{}})
}

func newGRPCServer(opts ...server.Option) server.Server {
	options := newOptions(opts...)

	// create a grpc server
	srv := &grpcServer{
		opts: options,
		rpc: &rServer{
			serviceMap: make(map[string]*service),
			logger:     options.Logger,
		},
		handlers:    make(map[string]server.Handler),
		subscribers: make(map[*subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
		wg:          wait(options.Context),
	}

	// configure the grpc server
	srv.configure()

	return srv
}

type grpcRouter struct {
	h func(context.Context, server.Request, interface{}) error
	m func(context.Context, server.Message) error
}

func (r grpcRouter) ProcessMessage(ctx context.Context, msg server.Message) error {
	return r.m(ctx, msg)
}

func (r grpcRouter) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	return r.h(ctx, req, rsp)
}

func (g *grpcServer) configure(opts ...server.Option) {
	g.Lock()
	defer g.Unlock()

	// Don't reprocess where there's no config
	if len(opts) == 0 && g.srv != nil {
		return
	}

	// Optionally use injected grpc.Server if there's a one
	var srv *grpc.Server
	if srv = g.getGrpcServer(); srv != nil {
		g.srv = srv
	}

	for _, o := range opts {
		o(&g.opts)
	}

	g.rsvc = nil

	// NOTE: injected grpc.Server doesn't have g.handler registered
	if srv != nil {
		return
	}

	maxMsgSize := g.getMaxMsgSize()

	gopts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.UnknownServiceHandler(g.handler),
	}

	if creds := g.getCredentials(); creds != nil {
		gopts = append(gopts, grpc.Creds(creds))
	}

	if opts := g.getGrpcOptions(); opts != nil {
		gopts = append(gopts, opts...)
	}

	g.srv = grpc.NewServer(gopts...)
}

func (g *grpcServer) getMaxMsgSize() int {
	if g.opts.Context == nil {
		return DefaultMaxMsgSize
	}
	s, ok := g.opts.Context.Value(maxMsgSizeKey{}).(int)
	if !ok {
		return DefaultMaxMsgSize
	}
	return s
}

func (g *grpcServer) getCredentials() credentials.TransportCredentials {
	if g.opts.Context != nil {
		if v, ok := g.opts.Context.Value(tlsAuth{}).(*tls.Config); ok && v != nil {
			return credentials.NewTLS(v)
		}
	}
	return nil
}

func (g *grpcServer) getGrpcOptions() []grpc.ServerOption {
	if g.opts.Context == nil {
		return nil
	}

	opts, ok := g.opts.Context.Value(grpcOptions{}).([]grpc.ServerOption)
	if !ok || opts == nil {
		return nil
	}

	return opts
}

func (g *grpcServer) getListener() net.Listener {
	if g.opts.Context == nil {
		return nil
	}

	if l, ok := g.opts.Context.Value(netListener{}).(net.Listener); ok && l != nil {
		return l
	}

	return nil
}

func (g *grpcServer) getGrpcServer() *grpc.Server {
	if g.opts.Context == nil {
		return nil
	}

	if srv, ok := g.opts.Context.Value(grpcServerKey{}).(*grpc.Server); ok && srv != nil {
		return srv
	}

	return nil
}

func (g *grpcServer) handler(srv interface{}, stream grpc.ServerStream) error {
	if g.wg != nil {
		g.wg.Add(1)
		defer g.wg.Done()
	}

	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Errorf(codes.Internal, "method does not exist in context")
	}

	serviceName, methodName, err := mgrpc.ServiceMethod(fullMethod)
	if err != nil {
		return status.New(codes.InvalidArgument, err.Error()).Err()
	}

	// get grpc metadata
	gmd, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		gmd = metadata.MD{}
	}

	// copy the metadata to go-micro.metadata
	md := meta.Metadata{}
	for k, v := range gmd {
		md[k] = strings.Join(v, ", ")
	}

	// timeout for server deadline
	to := md["timeout"]

	// get content type
	ct := defaultContentType

	if ctype, ok := md["x-content-type"]; ok {
		ct = ctype
	}
	if ctype, ok := md["content-type"]; ok {
		ct = ctype
	}

	delete(md, "x-content-type")
	delete(md, "timeout")

	// create new context
	ctx := meta.NewContext(stream.Context(), md)

	// get peer from context
	if p, ok := peer.FromContext(stream.Context()); ok {
		md["Remote"] = p.Addr.String()
		ctx = peer.NewContext(ctx, p)
	}

	// set the timeout if we have it
	if len(to) > 0 {
		if n, err := strconv.ParseUint(to, 10, 64); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
			defer cancel()
		}
	}

	// process via router
	if g.opts.Router != nil {
		cc, err := g.newGRPCCodec(ct)
		if err != nil {
			return errors.InternalServerError("go.micro.server", err.Error())
		}
		codec := &grpcCodec{
			method:   fmt.Sprintf("%s.%s", serviceName, methodName),
			endpoint: fmt.Sprintf("%s.%s", serviceName, methodName),
			target:   g.opts.Name,
			s:        stream,
			c:        cc,
		}

		// create a client.Request
		request := &rpcRequest{
			service:     mgrpc.ServiceFromMethod(fullMethod),
			contentType: ct,
			method:      fmt.Sprintf("%s.%s", serviceName, methodName),
			codec:       codec,
			stream:      true,
		}

		response := &rpcResponse{
			header: make(map[string]string),
			codec:  codec,
		}

		// create a wrapped function
		handler := func(ctx context.Context, req server.Request, rsp interface{}) error {
			return g.opts.Router.ServeRequest(ctx, req, rsp.(server.Response))
		}

		// execute the wrapper for it
		for i := len(g.opts.HdlrWrappers); i > 0; i-- {
			handler = g.opts.HdlrWrappers[i-1](handler)
		}

		r := grpcRouter{h: handler}

		// serve the actual request using the request router
		if err := r.ServeRequest(ctx, request, response); err != nil {
			if _, ok := status.FromError(err); ok {
				return err
			}
			return status.Errorf(codes.Internal, "%v", err.Error())
		}

		return nil
	}

	// process the standard request flow
	g.rpc.mu.Lock()
	service := g.rpc.serviceMap[serviceName]
	g.rpc.mu.Unlock()

	if service == nil {
		return status.New(codes.Unimplemented, fmt.Sprintf("unknown service %s", serviceName)).Err()
	}

	mtype := service.method[methodName]
	if mtype == nil {
		return status.New(codes.Unimplemented, fmt.Sprintf("unknown service %s.%s", serviceName, methodName)).Err()
	}

	// process unary
	if !mtype.stream {
		return g.processRequest(stream, service, mtype, ct, ctx)
	}

	// process stream
	return g.processStream(stream, service, mtype, ct, ctx)
}

func (g *grpcServer) processRequest(stream grpc.ServerStream, service *service, mtype *methodType, ct string, ctx context.Context) error {
	for {
		var argv, replyv reflect.Value

		// Decode the argument value.
		argIsValue := false // if true, need to indirect before calling.
		if mtype.ArgType.Kind() == reflect.Ptr {
			argv = reflect.New(mtype.ArgType.Elem())
		} else {
			argv = reflect.New(mtype.ArgType)
			argIsValue = true
		}

		// Unmarshal request
		if err := stream.RecvMsg(argv.Interface()); err != nil {
			return err
		}

		if argIsValue {
			argv = argv.Elem()
		}

		// reply value
		replyv = reflect.New(mtype.ReplyType.Elem())

		function := mtype.method.Func
		var returnValues []reflect.Value

		cc, err := g.newGRPCCodec(ct)
		if err != nil {
			return errors.InternalServerError("go.micro.server", err.Error())
		}
		b, err := cc.Marshal(argv.Interface())
		if err != nil {
			return err
		}

		// create a client.Request
		r := &rpcRequest{
			service:     g.opts.Name,
			contentType: ct,
			method:      fmt.Sprintf("%s.%s", service.name, mtype.method.Name),
			body:        b,
			payload:     argv.Interface(),
		}

		// define the handler func
		fn := func(ctx context.Context, req server.Request, rsp interface{}) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Extract(ctx).Errorf("panic recovered: %v, stack: %s", r, string(debug.Stack()))
					err = errors.InternalServerError("go.micro.server", "panic recovered: %v", r)
				}
			}()
			returnValues = function.Call([]reflect.Value{service.rcvr, mtype.prepareContext(ctx), reflect.ValueOf(argv.Interface()), reflect.ValueOf(rsp)})

			// The return value for the method is an error.
			if rerr := returnValues[0].Interface(); rerr != nil {
				err = rerr.(error)
			}

			return err
		}

		// wrap the handler func
		for i := len(g.opts.HdlrWrappers); i > 0; i-- {
			fn = g.opts.HdlrWrappers[i-1](fn)
		}
		statusCode := codes.OK
		statusDesc := ""
		// execute the handler
		if appErr := fn(ctx, r, replyv.Interface()); appErr != nil {
			var errStatus *status.Status
			switch verr := appErr.(type) {
			case *errors.Error:
				// micro.Error now proto based and we can attach it to grpc status
				statusCode = microError(verr)
				statusDesc = verr.Error()
				verr.Detail = strings.ToValidUTF8(verr.Detail, "")
				errStatus, err = status.New(statusCode, statusDesc).WithDetails(verr)
				if err != nil {
					return err
				}
			case proto.Message:
				// user defined error that proto based we can attach it to grpc status
				statusCode = convertCode(appErr)
				statusDesc = appErr.Error()
				errStatus, err = status.New(statusCode, statusDesc).WithDetails(verr)
				if err != nil {
					return err
				}
			default:
				// default case user pass own error type that not proto based
				statusCode = convertCode(verr)
				statusDesc = verr.Error()
				errStatus = status.New(statusCode, statusDesc)
			}

			return errStatus.Err()
		}

		if err := stream.SendMsg(replyv.Interface()); err != nil {
			return err
		}
		return status.New(statusCode, statusDesc).Err()
	}
}

func (g *grpcServer) processStream(stream grpc.ServerStream, service *service, mtype *methodType, ct string, ctx context.Context) error {
	opts := g.opts

	r := &rpcRequest{
		service:     opts.Name,
		contentType: ct,
		method:      fmt.Sprintf("%s.%s", service.name, mtype.method.Name),
		stream:      true,
	}

	ss := &rpcStream{
		request: r,
		s:       stream,
	}

	function := mtype.method.Func
	var returnValues []reflect.Value

	// Invoke the method, providing a new value for the reply.
	fn := func(ctx context.Context, req server.Request, stream interface{}) error {
		returnValues = function.Call([]reflect.Value{service.rcvr, mtype.prepareContext(ctx), reflect.ValueOf(stream)})
		if err := returnValues[0].Interface(); err != nil {
			return err.(error)
		}

		return nil
	}

	for i := len(opts.HdlrWrappers); i > 0; i-- {
		fn = opts.HdlrWrappers[i-1](fn)
	}

	statusCode := codes.OK
	statusDesc := ""

	if appErr := fn(ctx, r, ss); appErr != nil {
		var err error
		var errStatus *status.Status
		switch verr := appErr.(type) {
		case *errors.Error:
			// micro.Error now proto based and we can attach it to grpc status
			statusCode = microError(verr)
			statusDesc = verr.Error()
			verr.Detail = strings.ToValidUTF8(verr.Detail, "")
			errStatus, err = status.New(statusCode, statusDesc).WithDetails(verr)
			if err != nil {
				return err
			}
		case proto.Message:
			// user defined error that proto based we can attach it to grpc status
			statusCode = convertCode(appErr)
			statusDesc = appErr.Error()
			errStatus, err = status.New(statusCode, statusDesc).WithDetails(verr)
			if err != nil {
				return err
			}
		default:
			// default case user pass own error type that not proto based
			statusCode = convertCode(verr)
			statusDesc = verr.Error()
			errStatus = status.New(statusCode, statusDesc)
		}
		return errStatus.Err()
	}

	return status.New(statusCode, statusDesc).Err()
}

func (g *grpcServer) newGRPCCodec(contentType string) (encoding.Codec, error) {
	codecs := make(map[string]encoding.Codec)
	if g.opts.Context != nil {
		if v, ok := g.opts.Context.Value(codecsKey{}).(map[string]encoding.Codec); ok && v != nil {
			codecs = v
		}
	}
	if c, ok := codecs[contentType]; ok {
		return c, nil
	}
	if c, ok := defaultGRPCCodecs[contentType]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (g *grpcServer) Options() server.Options {
	g.RLock()
	opts := g.opts
	g.RUnlock()

	return opts
}

func (g *grpcServer) Init(opts ...server.Option) error {
	g.configure(opts...)
	return nil
}

func (g *grpcServer) NewHandler(h interface{}, opts ...server.HandlerOption) server.Handler {
	return newRpcHandler(h, opts...)
}

func (g *grpcServer) Handle(h server.Handler) error {
	if err := g.rpc.register(h.Handler()); err != nil {
		return err
	}

	g.handlers[h.Name()] = h
	return nil
}

func (g *grpcServer) NewSubscriber(topic string, sb interface{}, opts ...server.SubscriberOption) server.Subscriber {
	return newSubscriber(topic, sb, opts...)
}

func (g *grpcServer) Subscribe(sb server.Subscriber) error {
	sub, ok := sb.(*subscriber)
	if !ok {
		return fmt.Errorf("invalid subscriber: expected *subscriber")
	}
	if len(sub.handlers) == 0 {
		return fmt.Errorf("invalid subscriber: no handler functions")
	}

	if err := validateSubscriber(sb); err != nil {
		return err
	}

	g.Lock()
	if _, ok = g.subscribers[sub]; ok {
		g.Unlock()
		return fmt.Errorf("subscriber %v already exists", sub)
	}

	g.subscribers[sub] = nil
	g.Unlock()
	return nil
}

func (g *grpcServer) Register() error {
	g.RLock()
	rsvc := g.rsvc
	config := g.opts
	g.RUnlock()

	log := g.opts.Logger

	regFunc := func(service *registry.Service) error {
		var regErr error

		for i := 0; i < 3; i++ {
			// set the ttl
			rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}
			// attempt to register
			if err := config.Registry.Register(service, rOpts...); err != nil {
				// set the error
				regErr = err
				// backoff then retry
				time.Sleep(backoff.Do(i + 1))
				continue
			}
			// success so nil error
			regErr = nil
			break
		}

		return regErr
	}

	// if service already filled, reuse it and return early
	if rsvc != nil {
		if err := regFunc(rsvc); err != nil {
			return err
		}
		return nil
	}

	var err error
	var advt, host, port string
	var cacheService bool

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	if ip := net.ParseIP(host); ip != nil {
		cacheService = true
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// make copy of metadata
	md := meta.Copy(config.Metadata)

	// register service
	node := &registry.Node{
		Id:       config.Name + "-" + config.Id,
		Address:  mnet.HostPort(addr, port),
		Metadata: md,
	}

	node.Metadata["broker"] = config.Broker.String()
	node.Metadata["registry"] = config.Registry.String()
	node.Metadata["server"] = g.String()
	node.Metadata["transport"] = g.String()
	node.Metadata["protocol"] = "grpc"

	g.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, e := range g.handlers {
		// Only advertise non internal handlers
		if !e.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	var subscriberList []*subscriber
	for e := range g.subscribers {
		// Only advertise non internal subscribers
		if !e.Options().Internal {
			subscriberList = append(subscriberList, e)
		}
	}
	sort.Slice(subscriberList, func(i, j int) bool {
		return subscriberList[i].topic > subscriberList[j].topic
	})

	endpoints := make([]*registry.Endpoint, 0, len(handlerList)+len(subscriberList))
	for _, n := range handlerList {
		endpoints = append(endpoints, g.handlers[n].Endpoints()...)
	}
	for _, e := range subscriberList {
		endpoints = append(endpoints, e.Endpoints()...)
	}
	g.RUnlock()

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	g.RLock()
	registered := g.registered
	g.RUnlock()

	if !registered {
		log.Logf(logger.InfoLevel, "Registry [%s] Registering node: %s", config.Registry.String(), node.Id)
	}

	// register the service
	if err := regFunc(service); err != nil {
		return err
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	g.Lock()
	defer g.Unlock()

	for sb := range g.subscribers {
		handler := g.createSubHandler(sb, g.opts)
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}

		if cx := sb.Options().Context; cx != nil {
			opts = append(opts, broker.SubscribeContext(cx))
		}

		if !sb.Options().AutoAck {
			opts = append(opts, broker.DisableAutoAck())
		}

		log.Logf(logger.InfoLevel, "Subscribing to topic: %s", sb.Topic())

		sub, err := config.Broker.Subscribe(sb.Topic(), handler, opts...)
		if err != nil {
			return err
		}
		g.subscribers[sb] = []broker.Subscriber{sub}
	}

	g.registered = true
	if cacheService {
		g.rsvc = service
	}

	return nil
}

func (g *grpcServer) Deregister() error {
	var err error
	var advt, host, port string

	g.RLock()
	config := g.opts
	g.RUnlock()

	log := g.opts.Logger

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	node := &registry.Node{
		Id:      config.Name + "-" + config.Id,
		Address: mnet.HostPort(addr, port),
	}

	service := &registry.Service{
		Name:    config.Name,
		Version: config.Version,
		Nodes:   []*registry.Node{node},
	}

	log.Logf(logger.InfoLevel, "Deregistering node: %s", node.Id)

	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	g.Lock()
	g.rsvc = nil

	if !g.registered {
		g.Unlock()
		return nil
	}

	g.registered = false

	wg := sync.WaitGroup{}
	for sb, subs := range g.subscribers {
		for _, sub := range subs {
			wg.Add(1)
			go func(s broker.Subscriber) {
				defer wg.Done()
				log.Logf(logger.InfoLevel, "Unsubscribing from topic: %s", s.Topic())
				s.Unsubscribe()
			}(sub)
		}
		g.subscribers[sb] = nil
	}
	wg.Wait()

	g.Unlock()
	return nil
}

func (g *grpcServer) Start() error {
	g.RLock()
	if g.started {
		g.RUnlock()
		return nil
	}
	g.RUnlock()

	config := g.Options()
	log := config.Logger

	// micro: config.Transport.Listen(config.Address)
	var (
		ts  net.Listener
		err error
	)

	if l := g.getListener(); l != nil {
		ts = l
	} else {
		// check the tls config for secure connect
		if tc := config.TLSConfig; tc != nil {
			ts, err = tls.Listen("tcp", config.Address, tc)
			// otherwise just plain tcp listener
		} else {
			ts, err = net.Listen("tcp", config.Address)
		}
		if err != nil {
			return err
		}
	}

	if g.opts.Context != nil {
		if c, ok := g.opts.Context.Value(maxConnKey{}).(int); ok && c > 0 {
			ts = netutil.LimitListener(ts, c)
		}
	}

	log.Logf(logger.InfoLevel, "Server [grpc] Listening on %s", ts.Addr().String())
	g.Lock()
	g.opts.Address = ts.Addr().String()
	g.Unlock()

	// only connect if we're subscribed
	if len(g.subscribers) > 0 {
		// connect to the broker
		if err := config.Broker.Connect(); err != nil {
			log.Logf(logger.ErrorLevel, "Broker [%s] connect error: %v", config.Broker.String(), err)
			return err
		}

		log.Logf(logger.InfoLevel, "Broker [%s] Connected to %s", config.Broker.String(), config.Broker.Address())
	}

	// use RegisterCheck func before register
	if err = g.opts.RegisterCheck(g.opts.Context); err != nil {
		log.Logf(logger.ErrorLevel, "Server %s-%s register check error: %s", config.Name, config.Id, err)
	} else {
		// announce self to the world
		if err := g.Register(); err != nil {
			log.Logf(logger.ErrorLevel, "Server register error: %v", err)
		}
	}

	// micro: go ts.Accept(s.accept)
	go func() {
		if err := g.srv.Serve(ts); err != nil {
			log.Logf(logger.ErrorLevel, "gRPC Server start error: %v", err)
		}
	}()

	go func() {
		t := new(time.Ticker)

		// only process if it exists
		if g.opts.RegisterInterval > time.Duration(0) {
			// new ticker
			t = time.NewTicker(g.opts.RegisterInterval)
		}

		// return error chan
		var (
			err error
			ch  chan error
		)

	Loop:
		for {
			select {
			// register self on interval
			case <-t.C:
				g.RLock()
				registered := g.registered
				g.RUnlock()
				rerr := g.opts.RegisterCheck(g.opts.Context)
				if rerr != nil && registered {
					log.Logf(logger.ErrorLevel, "Server %s-%s register check error: %s, deregister it", config.Name, config.Id, rerr)
					// deregister self in case of error
					if err := g.Deregister(); err != nil {
						log.Logf(logger.ErrorLevel, "Server %s-%s deregister error: %s", config.Name, config.Id, err)
					}
				} else if rerr != nil && !registered {
					log.Logf(logger.ErrorLevel, "Server %s-%s register check error: %s", config.Name, config.Id, rerr)
					continue
				}
				if err := g.Register(); err != nil {
					log.Log(logger.ErrorLevel, "Server register error: ", err)
				}
			// wait for exit
			case ch = <-g.exit:
				break Loop
			}
		}

		// deregister self
		if err := g.Deregister(); err != nil {
			log.Log(logger.ErrorLevel, "Server deregister error: ", err)
		}

		// wait for waitgroup
		if g.wg != nil {
			g.wg.Wait()
		}

		// stop the grpc server
		exit := make(chan bool)

		go func() {
			g.srv.GracefulStop()
			close(exit)
		}()

		select {
		case <-exit:
		case <-time.After(time.Second):
			g.srv.Stop()
		}

		log.Logf(logger.InfoLevel, "Broker [%s] Disconnected from %s", config.Broker.String(), config.Broker.Address())
		// disconnect broker
		if err = config.Broker.Disconnect(); err != nil {
			log.Logf(logger.ErrorLevel, "Broker [%s] disconnect error: %v", config.Broker.String(), err)
		}

		// close transport
		ch <- err
	}()

	// mark the server as started
	g.Lock()
	g.started = true
	g.Unlock()

	return nil
}

func (g *grpcServer) Stop() error {
	g.RLock()
	if !g.started {
		g.RUnlock()
		return nil
	}
	g.RUnlock()

	ch := make(chan error)
	g.exit <- ch

	var err error
	select {
	case err = <-ch:
		g.Lock()
		g.rsvc = nil
		g.started = false
		g.Unlock()
	}

	return err
}

func (g *grpcServer) String() string {
	return "grpc"
}

func NewServer(opts ...server.Option) server.Server {
	return newGRPCServer(opts...)
}
