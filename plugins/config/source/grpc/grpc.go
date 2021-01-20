package grpc

import (
	"context"
	"crypto/tls"

	"github.com/asim/go-micro/v3/config/source"
	proto "github.com/asim/go-micro/plugins/config/source/grpc/v3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type grpcSource struct {
	addr      string
	path      string
	opts      source.Options
	tlsConfig *tls.Config
	client    *grpc.ClientConn
}

var (
	DefaultPath    = "/micro/config"
	DefaultAddress = "localhost:8080"
)

func (g *grpcSource) Read() (set *source.ChangeSet, err error) {

	var opts []grpc.DialOption

	// check if secure is necessary
	if g.tlsConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(g.tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	g.client, err = grpc.Dial(g.addr, opts...)
	if err != nil {
		return nil, err
	}
	cl := proto.NewSourceClient(g.client)
	rsp, err := cl.Read(context.Background(), &proto.ReadRequest{
		Path: g.path,
	})
	if err != nil {
		return nil, err
	}
	return toChangeSet(rsp.ChangeSet), nil
}

func (g *grpcSource) Watch() (source.Watcher, error) {
	cl := proto.NewSourceClient(g.client)
	rsp, err := cl.Watch(context.Background(), &proto.WatchRequest{
		Path: g.path,
	})
	if err != nil {
		return nil, err
	}
	return newWatcher(rsp)
}

// Write is unsupported
func (g *grpcSource) Write(cs *source.ChangeSet) error {
	return nil
}

func (g *grpcSource) String() string {
	return "grpc"
}

func NewSource(opts ...source.Option) source.Source {
	var options source.Options
	for _, o := range opts {
		o(&options)
	}

	addr := DefaultAddress
	path := DefaultPath

	if options.Context != nil {
		a, ok := options.Context.Value(addressKey{}).(string)
		if ok {
			addr = a
		}
		p, ok := options.Context.Value(pathKey{}).(string)
		if ok {
			path = p
		}
	}

	return &grpcSource{
		addr: addr,
		path: path,
		opts: options,
	}
}
