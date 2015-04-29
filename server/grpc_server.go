package server

import (
	"net"
	"sync"

	log "github.com/golang/glog"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	mtx     sync.RWMutex
	rpc     *grpc.Server
	address string
	exit    chan chan error
}

func (s *GRPCServer) Address() string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.address
}

func (s *GRPCServer) Init() error {
	return nil
}

func (s *GRPCServer) Register(handler HandlerFunc) error {
	handler(s)
	return nil
}

func (s *GRPCServer) Start() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	log.Infof("Listening on %s", l.Addr().String())

	s.mtx.Lock()
	s.address = l.Addr().String()
	s.mtx.Unlock()

	go s.rpc.Serve(l)

	go func() {
		ch := <-s.exit
		ch <- l.Close()
	}()

	return nil
}

func (s *GRPCServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}

func NewGRPCServer(address string) *GRPCServer {
	return &GRPCServer{
		rpc:     grpc.NewServer(),
		address: address,
		exit:    make(chan chan error),
	}
}

type GRPCHandlerFunc func(s *grpc.Server)

func GRPCHandler(f GRPCHandlerFunc) HandlerFunc {
	return func(srv Server) {
		f(srv.(*GRPCServer).rpc)
	}
}
