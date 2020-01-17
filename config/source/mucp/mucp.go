package mucp

import (
	"context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/config/source"
	proto "github.com/micro/go-micro/config/source/mucp/proto"
	"github.com/micro/go-micro/util/log"
)

var (
	DefaultServiceName = "go.micro.config"
	DefaultClient      = client.DefaultClient
)

type service struct {
	serviceName string
	key         string
	opts        source.Options
	client      proto.Service
}

func (m *service) Read() (set *source.ChangeSet, err error) {
	req, err := m.client.Read(context.Background(), &proto.ReadRequest{Path: m.key})
	if err != nil {
		return nil, err
	}

	return toChangeSet(req.Change.ChangeSet), nil
}

func (m *service) Watch() (w source.Watcher, err error) {
	stream, err := m.client.Watch(context.Background(), &proto.WatchRequest{Key: m.key})
	if err != nil {
		log.Error("watch err: ", err)
		return
	}
	return newWatcher(stream)
}

// Write is unsupported
func (m *service) Write(cs *source.ChangeSet) error {
	return nil
}

func (m *service) String() string {
	return "mucp"
}

func NewSource(opts ...source.Option) source.Source {
	var options source.Options
	for _, o := range opts {
		o(&options)
	}

	addr := DefaultServiceName

	if options.Context != nil {
		a, ok := options.Context.Value(serviceNameKey{}).(string)
		if ok {
			addr = a
		}
	}

	s := &service{
		serviceName: addr,
		opts:        options,
		client:      proto.NewService(addr, DefaultClient),
	}

	return s
}
