package service

import (
	"context"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/config/source"
	proto "github.com/micro/go-micro/config/source/service/proto"
	"github.com/micro/go-micro/util/log"
)

var (
	DefaultName   = "go.micro.config"
	DefaultKey    = "NAMESPACE:CONFIG"
	DefaultPath   = ""
	DefaultClient = client.DefaultClient
)

type service struct {
	serviceName string
	key         string
	path        string
	opts        source.Options
	client      proto.ConfigService
}

func (m *service) Read() (set *source.ChangeSet, err error) {
	req, err := m.client.Read(context.Background(), &proto.ReadRequest{Key: m.key, Path: m.path})
	if err != nil {
		return nil, err
	}

	return toChangeSet(req.Change.ChangeSet), nil
}

func (m *service) Watch() (w source.Watcher, err error) {
	stream, err := m.client.Watch(context.Background(), &proto.WatchRequest{Key: m.key, Path: m.path})
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
	return "service"
}

func NewSource(opts ...source.Option) source.Source {
	var options source.Options
	for _, o := range opts {
		o(&options)
	}

	addr := DefaultName
	key := DefaultKey
	path := DefaultPath

	if options.Context != nil {
		a, ok := options.Context.Value(serviceNameKey{}).(string)
		if ok {
			addr = a
		}

		k, ok := options.Context.Value(keyKey{}).(string)
		if ok {
			key = k
		}

		p, ok := options.Context.Value(pathKey{}).(string)
		if ok {
			path = p
		}
	}

	s := &service{
		serviceName: addr,
		opts:        options,
		key:         key,
		path:        path,
		client:      proto.NewConfigService(addr, DefaultClient),
	}

	return s
}
