package etcd

import (
	"context"
	"errors"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/micro/go-micro/v2/registry"
)

type etcdWatcher struct {
	stop    chan bool
	w       clientv3.WatchChan
	client  *clientv3.Client
	timeout time.Duration
}

func newEtcdWatcher(r *etcdRegistry, timeout time.Duration, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	if len(wo.Domain) == 0 {
		wo.Domain = r.options.Domain
	}

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan bool, 1)

	go func() {
		<-stop
		cancel()
	}()

	watchPath := prefix
	if len(wo.Service) > 0 {
		watchPath = servicePath(wo.Domain, wo.Service) + "/"
	}

	return &etcdWatcher{
		stop:    stop,
		w:       r.client.Watch(ctx, watchPath, clientv3.WithPrefix(), clientv3.WithPrevKV()),
		client:  r.client,
		timeout: timeout,
	}, nil
}

func (ew *etcdWatcher) Chan() chan *registry.Result {
	c := make(chan *registry.Result)

	go func() {
		for wresp := range ew.w {
			if wresp.Err() != nil || wresp.Canceled {
				close(c)
				return
			}

			for _, ev := range wresp.Events {
				if r := serializeResult(ev); r != nil {
					c <- r
				}
			}
		}

		close(c)
	}()

	return c
}

func (ew *etcdWatcher) Next() (*registry.Result, error) {
	for wresp := range ew.w {
		if wresp.Err() != nil {
			return nil, wresp.Err()
		}
		if wresp.Canceled {
			return nil, errors.New("could not get next")
		}
		for _, ev := range wresp.Events {
			if r := serializeResult(ev); r != nil {
				return r, nil
			}
		}
	}
	return nil, errors.New("could not get next")
}

func (ew *etcdWatcher) Stop() {
	select {
	case <-ew.stop:
		return
	default:
		close(ew.stop)
	}
}

func serializeResult(ev *clientv3.Event) *registry.Result {
	service := decode(ev.Kv.Value)
	var action string

	switch ev.Type {
	case clientv3.EventTypePut:
		if ev.IsCreate() {
			action = "create"
		} else if ev.IsModify() {
			action = "update"
		}
	case clientv3.EventTypeDelete:
		action = "delete"

		// get service from prevKv
		service = decode(ev.PrevKv.Value)
	}

	if service == nil {
		return nil
	}

	return &registry.Result{
		Action:  action,
		Service: service,
	}
}
