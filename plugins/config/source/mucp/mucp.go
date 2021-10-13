package mucp

import (
	"context"

	"go-micro.dev/v4/cmd"
	"go-micro.dev/v4/config/source"
	log "go-micro.dev/v4/logger"
	proto "github.com/asim/go-micro/plugins/config/source/mucp/v4/proto"
)

var (
	DefaultPath        = "/micro/config"
	DefaultServiceName = "go.micro.config"
)

type mucpSource struct {
	serviceName string
	path        string
	opts        source.Options
	client      proto.SourceService
}

func (m *mucpSource) Read() (set *source.ChangeSet, err error) {
	req, err := m.client.Read(context.Background(), &proto.ReadRequest{Path: m.path})
	if err != nil {
		return nil, err
	}

	return toChangeSet(req.ChangeSet), nil
}

func (m *mucpSource) Watch() (w source.Watcher, err error) {
	stream, err := m.client.Watch(context.Background(), &proto.WatchRequest{Path: m.path})
	if err != nil {
		log.Error("watch err: ", err)
		return
	}
	return newWatcher(stream)
}

// Write is unsupported
func (m *mucpSource) Write(cs *source.ChangeSet) error {
	return nil
}

func (m *mucpSource) String() string {
	return "mucp"
}

func NewSource(opts ...source.Option) source.Source {
	var options source.Options
	for _, o := range opts {
		o(&options)
	}

	addr := DefaultServiceName
	path := DefaultPath

	if options.Context != nil {
		a, ok := options.Context.Value(serviceNameKey{}).(string)
		if ok {
			addr = a
		}
		p, ok := options.Context.Value(pathKey{}).(string)
		if ok {
			path = p
		}
	}

	s := &mucpSource{
		serviceName: addr,
		path:        path,
		opts:        options,
		client:      proto.NewSourceService(addr, *cmd.DefaultOptions().Client),
	}

	return s
}
