package etcd

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/micro/go-micro/v3/registry"
)

type etcdWatcher struct {
	w       clientv3.WatchChan
	client  *clientv3.Client
	timeout time.Duration

	mtx    sync.Mutex
	stop   chan bool
	cancel func()
}

func newEtcdWatcher(c *clientv3.Client, timeout time.Duration, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	if len(wo.Domain) == 0 {
		wo.Domain = defaultDomain
	}

	watchPath := prefix
	if wo.Domain == registry.WildcardDomain {
		if len(wo.Service) > 0 {
			return nil, errors.New("Cannot watch a service across domains")
		}
		watchPath = prefix
	} else if len(wo.Service) > 0 {
		watchPath = servicePath(wo.Domain, wo.Service) + "/"
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := c.Watch(ctx, watchPath, clientv3.WithPrefix(), clientv3.WithPrevKV())
	stop := make(chan bool, 1)

	return &etcdWatcher{
		cancel:  cancel,
		stop:    stop,
		w:       w,
		client:  c,
		timeout: timeout,
	}, nil
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
				continue
			}
			return &registry.Result{
				Action:  action,
				Service: service,
			}, nil
		}
	}
	return nil, errors.New("could not get next")
}

func (ew *etcdWatcher) Stop() {
	ew.mtx.Lock()
	defer ew.mtx.Unlock()

	select {
	case <-ew.stop:
		return
	default:
		close(ew.stop)
		ew.cancel()
		ew.client.Close()
	}
}
