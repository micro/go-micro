package server

import (
	"bytes"
	"sync"

	c "github.com/myodc/go-micro/context"
	"github.com/myodc/go-micro/transport"

	log "github.com/golang/glog"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"

	"golang.org/x/net/context"
)

type rpcServer struct {
	mtx     sync.RWMutex
	address string
	opts    options
	rpc     *rpc.Server
	exit    chan chan error
}

func newRpcServer(address string, opt ...Option) Server {
	var opts options

	for _, o := range opt {
		o(&opts)
	}

	if opts.transport == nil {
		opts.transport = transport.DefaultTransport
	}

	return &rpcServer{
		opts:    opts,
		address: address,
		rpc:     rpc.NewServer(),
		exit:    make(chan chan error),
	}
}

func (s *rpcServer) accept(sock transport.Socket) {
	var msg transport.Message
	if err := sock.Recv(&msg); err != nil {
		return
	}

	rbq := bytes.NewBuffer(msg.Body)
	rsp := bytes.NewBuffer(nil)
	defer rsp.Reset()
	defer rbq.Reset()

	buf := &buffer{
		rbq,
		rsp,
	}

	var cc rpc.ServerCodec
	switch msg.Header["Content-Type"] {
	case "application/octet-stream":
		cc = pb.NewServerCodec(buf)
	case "application/json":
		cc = js.NewServerCodec(buf)
	default:
		return
	}

	// strip our headers
	ct := msg.Header["Content-Type"]
	delete(msg.Header, "Content-Type")

	ctx := c.WithMetaData(context.Background(), msg.Header)

	if err := s.rpc.ServeRequestWithContext(ctx, cc); err != nil {
		return
	}

	sock.Send(&transport.Message{
		Header: map[string]string{
			"Content-Type": ct,
		},
		Body: rsp.Bytes(),
	})
}

func (s *rpcServer) Address() string {
	s.mtx.RLock()
	address := s.address
	s.mtx.RUnlock()
	return address
}

func (s *rpcServer) Init() error {
	return nil
}

func (s *rpcServer) NewReceiver(handler interface{}) Receiver {
	return newRpcReceiver("", handler)
}

func (s *rpcServer) NewNamedReceiver(name string, handler interface{}) Receiver {
	return newRpcReceiver(name, handler)
}

func (s *rpcServer) Register(r Receiver) error {
	if len(r.Name()) > 0 {
		s.rpc.RegisterName(r.Name(), r.Handler())
		return nil
	}

	s.rpc.Register(r.Handler())
	return nil
}

func (s *rpcServer) Start() error {
	registerHealthChecker(s)

	ts, err := s.opts.transport.Listen(s.address)
	if err != nil {
		return err
	}

	log.Infof("Listening on %s", ts.Addr())

	s.mtx.RLock()
	s.address = ts.Addr()
	s.mtx.RUnlock()

	go ts.Accept(s.accept)

	go func() {
		ch := <-s.exit
		ch <- ts.Close()
	}()

	return nil
}

func (s *rpcServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}
