package nacos

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"sync"

	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type watcher struct {
	n  *nacosRegistry
	wo registry.WatchOptions

	next chan *registry.Result
	exit chan bool

	sync.RWMutex
	services      map[string][]*registry.Service
	cacheServices map[string][]model.Instance
	param         *vo.SubscribeParam
	Doms          []string
}

func newWatcher(nr *nacosRegistry, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	nw := watcher{
		n:             nr,
		wo:            wo,
		exit:          make(chan bool),
		next:          make(chan *registry.Result, 10),
		services:      make(map[string][]*registry.Service),
		cacheServices: make(map[string][]model.Instance),
		param:         new(vo.SubscribeParam),
		Doms:          make([]string, 0),
	}
	withContext := false
	if wo.Context != nil {
		if p, ok := wo.Context.Value("subscribe_param").(vo.SubscribeParam); ok {
			nw.param = &p
			withContext = ok
			nw.param.SubscribeCallback = nw.callBackHandle
			go nr.client.Subscribe(nw.param)
		}
	}
	if !withContext {
		param := vo.GetAllServiceInfoParam{}
		services, err := nr.client.GetAllServicesInfo(param)
		if err != nil {
			return nil, err
		}
		param.PageNo = 1
		param.PageSize = uint32(services.Count)
		services, err = nr.client.GetAllServicesInfo(param)
		if err != nil {
			return nil, err
		}
		nw.Doms = services.Doms
		for _, v := range nw.Doms {
			param := &vo.SubscribeParam{
				ServiceName:       v,
				SubscribeCallback: nw.callBackHandle,
			}
			go nr.client.Subscribe(param)
		}
	}

	return &nw, nil
}

func (nw *watcher) callBackHandle(services []model.Instance, err error) {
	if err != nil {
		logger.Error("nacos watcher call back handle error:%v", err)
		return
	}
	serviceName := services[0].ServiceName

	if nw.cacheServices[serviceName] == nil {

		nw.Lock()
		nw.cacheServices[serviceName] = services
		nw.Unlock()

		for _, v := range services {
			nw.next <- &registry.Result{Action: "create", Service: buildRegistryService(&v)}
			return
		}
	} else {
		for _, subscribeService := range services {
			create := true
			for _, cacheService := range nw.cacheServices[serviceName] {
				if subscribeService.InstanceId == cacheService.InstanceId {
					if !reflect.DeepEqual(subscribeService, cacheService) {
						//update instance
						nw.next <- &registry.Result{Action: "update", Service: buildRegistryService(&subscribeService)}
						return
					}
					create = false
				}
			}
			//new instance
			if create {
				log.Println("create", subscribeService.ServiceName, subscribeService.Port)

				nw.next <- &registry.Result{Action: "create", Service: buildRegistryService(&subscribeService)}

				nw.Lock()
				nw.cacheServices[serviceName] = append(nw.cacheServices[serviceName], subscribeService)
				nw.Unlock()
				return
			}
		}

		for index, cacheService := range nw.cacheServices[serviceName] {
			del := true
			for _, subscribeService := range services {
				if subscribeService.InstanceId == cacheService.InstanceId {
					del = false
				}
			}
			if del {
				log.Println("del", cacheService.ServiceName, cacheService.Port)
				nw.next <- &registry.Result{Action: "delete", Service: buildRegistryService(&cacheService)}

				nw.Lock()
				nw.cacheServices[serviceName][index] = model.Instance{}
				nw.Unlock()

				return
			}
		}
	}

}

func buildRegistryService(v *model.Instance) (s *registry.Service) {
	nodes := make([]*registry.Node, 0)
	nodes = append(nodes, &registry.Node{
		Id:       v.InstanceId,
		Address:  net.JoinHostPort(v.Ip, fmt.Sprintf("%d", v.Port)),
		Metadata: v.Metadata,
	})
	s = &registry.Service{
		Name:     v.ServiceName,
		Version:  "latest",
		Metadata: v.Metadata,
		Nodes:    nodes,
	}
	return
}

func (nw *watcher) Next() (r *registry.Result, err error) {
	select {
	case <-nw.exit:
		return nil, registry.ErrWatcherStopped
	case r, ok := <-nw.next:
		if !ok {
			return nil, registry.ErrWatcherStopped
		}
		return r, nil
	}
}

func (nw *watcher) Stop() {
	select {
	case <-nw.exit:
		return
	default:
		close(nw.exit)
		if len(nw.Doms) > 0 {
			for _, v := range nw.Doms {
				param := &vo.SubscribeParam{
					ServiceName:       v,
					SubscribeCallback: nw.callBackHandle,
				}
				_ = nw.n.client.Unsubscribe(param)
			}
		} else {
			_ = nw.n.client.Unsubscribe(nw.param)
		}
	}
}
