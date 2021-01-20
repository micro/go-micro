package grpc

import (
	"github.com/asim/go-micro/v3/config/source"
	proto "github.com/asim/go-micro/plugins/config/source/grpc/v3/proto"
)

type watcher struct {
	stream proto.Source_WatchClient
}

func newWatcher(stream proto.Source_WatchClient) (*watcher, error) {
	return &watcher{
		stream: stream,
	}, nil
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	rsp, err := w.stream.Recv()
	if err != nil {
		return nil, err
	}
	return toChangeSet(rsp.ChangeSet), nil
}

func (w *watcher) Stop() error {
	return w.stream.CloseSend()
}
