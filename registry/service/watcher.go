package service

import (
	"github.com/micro/go-micro/v2/registry"
	pb "github.com/micro/go-micro/v2/registry/service/proto"
)

type serviceWatcher struct {
	stream pb.Registry_WatchService
	closed chan bool
}

func (s *serviceWatcher) Next() (*registry.Result, error) {
	// check if closed
	select {
	case <-s.closed:
		return nil, registry.ErrWatcherStopped
	default:
	}

	r, err := s.stream.Recv()
	if err != nil {
		return nil, err
	}

	return &registry.Result{
		Action:  r.Action,
		Service: ToService(r.Service),
	}, nil
}

func (s *serviceWatcher) Stop() {
	select {
	case <-s.closed:
		return
	default:
		close(s.closed)
		s.stream.Close()
	}
}

func newWatcher(stream pb.Registry_WatchService) registry.Watcher {
	return &serviceWatcher{
		stream: stream,
		closed: make(chan bool),
	}
}
