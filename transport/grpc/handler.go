package grpc

import (
	"runtime/debug"

	"github.com/micro/go-micro/transport"
	pb "github.com/micro/go-micro/transport/grpc/proto"
	"github.com/micro/go-micro/util/log"
	"google.golang.org/grpc/peer"
)

// microTransport satisfies the pb.TransportServer inteface
type microTransport struct {
	addr string
	fn   func(transport.Socket)
}

func (m *microTransport) Stream(ts pb.Transport_StreamServer) error {
	sock := &grpcTransportSocket{
		stream: ts,
		local:  m.addr,
	}

	p, ok := peer.FromContext(ts.Context())
	if ok {
		sock.remote = p.Addr.String()
	}

	defer func() {
		if r := recover(); r != nil {
			log.Log(r, string(debug.Stack()))
			sock.Close()
		}
	}()

	// execute socket func
	m.fn(sock)
	return nil
}
