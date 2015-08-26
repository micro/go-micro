package etcd

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/coreos/go-etcd/etcd"
	"github.com/kynrai/go-micro/registry"
)

var (
	prefix = "/micro-registry"
)

type etcdRegistry struct {
	client *etcd.Client

	sync.RWMutex
	services map[string]*registry.Service
}

func encode(s *registry.Service) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(ds string) *registry.Service {
	var s *registry.Service
	json.Unmarshal([]byte(ds), &s)
	return s
}

func nodePath(s, id string) string {
	service := strings.Replace(s, "/", "-", -1)
	node := strings.Replace(id, "/", "-", -1)
	return filepath.Join(prefix, service, node)
}

func servicePath(s string) string {
	return filepath.Join(prefix, strings.Replace(s, "/", "-", -1))
}

func (e *etcdRegistry) Deregister(s *registry.Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	for _, node := range s.Nodes {
		_, err := e.client.Delete(nodePath(s.Name, node.Id), false)
		if err != nil {
			return err
		}
	}

	e.client.DeleteDir(servicePath(s.Name))
	return nil
}

func (e *etcdRegistry) Register(s *registry.Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	service := &registry.Service{
		Name:     s.Name,
		Metadata: s.Metadata,
	}

	e.client.CreateDir(servicePath(s.Name), 0)

	for _, node := range s.Nodes {
		service.Nodes = []*registry.Node{node}
		_, err := e.client.Create(nodePath(service.Name, node.Id), encode(service), 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *etcdRegistry) GetService(name string) (*registry.Service, error) {
	e.RLock()
	service, ok := e.services[name]
	e.RUnlock()

	if ok {
		return service, nil
	}

	rsp, err := e.client.Get(servicePath(name), false, false)
	if err != nil && !strings.HasPrefix(err.Error(), "100: Key not found") {
		return nil, err
	}

	s := &registry.Service{}

	for _, n := range rsp.Node.Nodes {
		if n.Dir {
			continue
		}
		sn := decode(n.Value)
		for _, node := range sn.Nodes {
			s.Nodes = append(s.Nodes, node)
		}
	}

	return s, nil
}

func (e *etcdRegistry) ListServices() ([]*registry.Service, error) {
	e.RLock()
	serviceMap := e.services
	e.RUnlock()

	var services []*registry.Service

	if len(serviceMap) > 0 {
		for _, service := range services {
			services = append(services, service)
		}
		return services, nil
	}

	rsp, err := e.client.Get(prefix, true, true)
	if err != nil && !strings.HasPrefix(err.Error(), "100: Key not found") {
		return nil, err
	}

	for _, node := range rsp.Node.Nodes {
		service := &registry.Service{}

		for _, n := range node.Nodes {
			i := decode(n.Value)
			service.Name = i.Name
			for _, in := range i.Nodes {
				service.Nodes = append(service.Nodes, in)
			}
		}

		services = append(services, service)
	}

	return services, nil
}

func (e *etcdRegistry) Watch() (registry.Watcher, error) {
	// todo: fix watcher
	return newEtcdWatcher(e)
}

func NewRegistry(addrs []string, opt ...registry.Option) registry.Registry {
	var cAddrs []string

	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}

	if len(cAddrs) == 0 {
		cAddrs = []string{"http://127.0.0.1:2379"}
	}

	e := &etcdRegistry{
		client:   etcd.NewClient(cAddrs),
		services: make(map[string]*registry.Service),
	}

	return e
}
