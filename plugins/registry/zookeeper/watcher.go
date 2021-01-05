package zookeeper

import (
	"errors"
	"path"

	"github.com/micro/go-micro/v2/registry"
	"github.com/samuel/go-zookeeper/zk"
)

type zookeeperWatcher struct {
	wo       registry.WatchOptions
	client   *zk.Conn
	stop     chan bool
	results  chan result
	respChan chan watchResponse
}

type watchResponse struct {
	event   zk.Event
	service *registry.Service
	err     error
}

type result struct {
	res *registry.Result
	err error
}

func newZookeeperWatcher(r *zookeeperRegistry, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	zw := &zookeeperWatcher{
		wo:       wo,
		client:   r.client,
		stop:     make(chan bool),
		results:  make(chan result),
		respChan: make(chan watchResponse),
	}

	go zw.watch()
	return zw, nil
}

func (zw *zookeeperWatcher) writeRespChan(resp *watchResponse) {
	if resp == nil {
		return
	}
	select {
	case <-zw.stop:
	default:
		zw.respChan <- *resp
	}
}

func (zw *zookeeperWatcher) writeResult(r *result) {
	if r == nil {
		return
	}
	select {
	case <-zw.stop:
	default:
		zw.results <- *r
	}
}

func (zw *zookeeperWatcher) watchDir(key string) {
	// mark children changed
	childrenChanged := true

	for {
		// get current children for a key
		newChildren, _, childEventCh, err := zw.client.ChildrenW(key)
		if err != nil {
			zw.writeRespChan(&watchResponse{zk.Event{}, nil, err})
			return
		}

		if childrenChanged {
			zw.overrideChildrenInfo(newChildren, key)
		}

		select {
		case e := <-childEventCh:
			if e.Type != zk.EventNodeChildrenChanged {
				childrenChanged = false
				continue
			}

			childrenChanged = true
		case <-zw.stop:
			// There is no way to stop GetW/ChildrenW so just quit
			return
		}
	}
}

func (zw *zookeeperWatcher) watchKey(key string) {
	for {
		s, _, keyEventCh, err := zw.client.GetW(key)
		if err != nil {
			zw.writeRespChan(&watchResponse{zk.Event{}, nil, err})
			return
		}

		select {
		case e := <-keyEventCh:
			switch e.Type {
			case zk.EventNodeDataChanged, zk.EventNodeCreated, zk.EventNodeDeleted:
				if e.Type != zk.EventNodeDeleted {
					// get the updated service
					s, _, err = zw.client.Get(e.Path)
					if err != nil {
						continue
					}
				}

				srv, err := decode(s)
				if err != nil {
					continue
				}

				zw.writeRespChan(&watchResponse{e, srv, err})
			}
			if e.Type == zk.EventNodeDeleted {
				//The Node was deleted - stop watching
				return
			}
		case <-zw.stop:
			// There is no way to stop GetW/ChildrenW so just quit
			return
		}
	}
}

func (zw *zookeeperWatcher) watch() {

	services := func() []string {
		if len(zw.wo.Service) > 0 {
			return []string{zw.wo.Service}
		}
		allServices, _, err := zw.client.Children(prefix)
		if err != nil {
			zw.writeResult(&result{nil, err})
		}
		return allServices
	}

	//watch every service
	for _, service := range services() {
		sPath := childPath(prefix, service)
		if sPath == prefix {
			continue
		}
		go zw.watchDir(sPath)
	}

	var service *registry.Service
	var action string
	for {
		select {
		case <-zw.stop:
			return
		case rsp := <-zw.respChan:
			if rsp.err != nil {
				zw.writeResult(&result{nil, rsp.err})
				continue
			}
			switch rsp.event.Type {
			case zk.EventNodeDataChanged:
				action = "update"
				service = rsp.service
			case zk.EventNodeDeleted:
				action = "delete"
				service = rsp.service
			case zk.EventNodeCreated:
				action = "create"
				service = rsp.service
			case zk.EventNodeChildrenChanged:
				action = "override"
				service = rsp.service
			}
		}
		zw.writeResult(&result{&registry.Result{Action: action, Service: service}, nil})
	}
}

func (zw *zookeeperWatcher) Stop() {
	select {
	case <-zw.stop:
		return
	default:
		close(zw.stop)
	}
}

func (zw *zookeeperWatcher) Next() (*registry.Result, error) {
	select {
	case <-zw.stop:
		return nil, errors.New("watcher stopped")
	case r := <-zw.results:
		return r.res, r.err
	}
}

func (zw *zookeeperWatcher) overrideChildrenInfo(newChildren []string, parentPath string) {
	// override resp
	var overrideResp *watchResponse

	for _, i := range newChildren {
		newNode := path.Join(parentPath, i)

		s, _, err := zw.client.Get(newNode)
		if err != nil {
			continue
		}

		srv, err := decode(s)
		if err != nil {
			continue
		}

		// if nil, then init, do it once
		if overrideResp == nil {
			overrideResp = &watchResponse{zk.Event{
				Type: zk.EventNodeChildrenChanged,
			}, srv, nil}
			zw.writeRespChan(overrideResp)
		}

		// when node was added or updated
		zw.writeRespChan(&watchResponse{zk.Event{
			Type: zk.EventNodeCreated,
		}, srv, nil})
	}
}
