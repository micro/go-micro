package server

import (
	"bytes"

	c "github.com/myodc/go-micro/context"
	"github.com/myodc/go-micro/transport"

	log "github.com/golang/glog"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"

	"golang.org/x/net/context"
)

type rpcServer struct {
	opts options
	rpc  *rpc.Server
	exit chan chan error
}

func newRpcServer(opts ...Option) Server {
	return &rpcServer{
		opts: newOptions(opts...),
		rpc:  rpc.NewServer(),
		exit: make(chan chan error),
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

	ctx := c.WithMetadata(context.Background(), msg.Header)

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

func (s *rpcServer) Config() options {
	return s.opts
}

func (s *rpcServer) Init(opts ...Option) {
	for _, opt := range opts {
		opt(&s.opts)
	}
	if len(s.opts.id) == 0 {
		s.opts.id = s.opts.name + "-" + DefaultId
	}
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

	ts, err := s.opts.transport.Listen(s.opts.address)
	if err != nil {
		return err
	}

	log.Infof("Listening on %s", ts.Addr())

	s.opts.address = ts.Addr()

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
