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
	for {
		// get current children for a key
		children, _, childEventCh, err := zw.client.ChildrenW(key)
		if err != nil {
			zw.writeRespChan(&watchResponse{zk.Event{}, nil, err})
			return
		}

		select {
		case e := <-childEventCh:
			if e.Type != zk.EventNodeChildrenChanged {
				continue
			}

			newChildren, _, err := zw.client.Children(e.Path)
			if err != nil {
				zw.writeRespChan(&watchResponse{e, nil, err})
				return
			}

			// a node was added -- watch the new node
			for _, i := range newChildren {
				if contains(children, i) {
					continue
				}

				newNode := path.Join(e.Path, i)

				if key == prefix {
					// a new service was created under prefix
					go zw.watchDir(newNode)

					nodes, _, _ := zw.client.Children(newNode)
					for _, node := range nodes {
						n := path.Join(newNode, node)
						go zw.watchKey(n)
						s, _, err := zw.client.Get(n)
						if err != nil {
							continue
						}
						e.Type = zk.EventNodeCreated

						srv, err := decode(s)
						if err != nil {
							continue
						}

						zw.writeRespChan(&watchResponse{e, srv, err})
					}
				} else {
					go zw.watchKey(newNode)
					s, _, err := zw.client.Get(newNode)
					if err != nil {
						continue
					}
					e.Type = zk.EventNodeCreated

					srv, err := decode(s)
					if err != nil {
						continue
					}

					zw.writeRespChan(&watchResponse{e, srv, err})
				}
			}
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
		go zw.watchDir(sPath)
		children, _, err := zw.client.Children(sPath)
		if err != nil {
			zw.writeResult(&result{nil, err})
		}
		for _, c := range children {
			go zw.watchKey(path.Join(sPath, c))
		}
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
