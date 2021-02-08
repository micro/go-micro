// Package zookeeper provides a zookeeper registry
package zookeeper

import (
	"errors"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/cmd"
	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/go-zookeeper/zk"
	hash "github.com/mitchellh/hashstructure"
)

var (
	prefix = "/micro-registry"
)

type zookeeperRegistry struct {
	client  *zk.Conn
	options registry.Options
	sync.Mutex
	register map[string]uint64
}

func init() {
	cmd.DefaultRegistries["zookeeper"] = NewRegistry
}

func configure(z *zookeeperRegistry, opts ...registry.Option) error {
	cAddrs := z.options.Addrs

	for _, o := range opts {
		o(&z.options)
	}

	if z.options.Timeout == 0 {
		z.options.Timeout = 5
	}

	// already set
	if z.client != nil && len(z.options.Addrs) == len(cAddrs) {
		return nil
	}

	// reset
	cAddrs = nil

	for _, addr := range z.options.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}

	if len(cAddrs) == 0 {
		cAddrs = []string{"127.0.0.1:2181"}
	}

	// connect to zookeeper
	c, _, err := zk.Connect(cAddrs, time.Second*z.options.Timeout)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// create our prefix path
	if err := createPath(prefix, []byte{}, c); err != nil {
		log.Error(err.Error())
		return err
	}

	z.client = c
	return nil
}

func (z *zookeeperRegistry) Init(opts ...registry.Option) error {
	return configure(z, opts...)
}

func (z *zookeeperRegistry) Options() registry.Options {
	return z.options
}

func (z *zookeeperRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	// delete our hash of the service
	z.Lock()
	delete(z.register, s.Name)
	z.Unlock()

	for _, node := range s.Nodes {
		err := z.client.Delete(nodePath(s.Name, node.Id), -1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (z *zookeeperRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	// create hash of service; uint64
	h, err := hash.Hash(s, nil)
	if err != nil {
		return err
	}

	// get existing hash
	z.Lock()
	v, ok := z.register[s.Name]
	z.Unlock()

	// the service is unchanged, skip registering
	if ok && v == h {
		return nil
	}

	service := &registry.Service{
		Name:      s.Name,
		Version:   s.Version,
		Metadata:  s.Metadata,
		Endpoints: s.Endpoints,
	}

	for _, node := range s.Nodes {
		service.Nodes = []*registry.Node{node}
		exists, _, err := z.client.Exists(nodePath(service.Name, node.Id))
		if err != nil {
			return err
		}

		srv, err := encode(service)
		if err != nil {
			return err
		}

		if exists {
			_, err := z.client.Set(nodePath(service.Name, node.Id), srv, -1)
			if err != nil {
				return err
			}
		} else {
			err := createPath(nodePath(service.Name, node.Id), srv, z.client)
			if err != nil {
				return err
			}
		}
	}

	// save our hash of the service
	z.Lock()
	z.register[s.Name] = h
	z.Unlock()

	return nil
}

func (z *zookeeperRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	l, _, err := z.client.Children(servicePath(name))
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*registry.Service)

	for _, n := range l {
		_, stat, err := z.client.Children(nodePath(name, n))
		if err != nil {
			return nil, err
		}

		if stat.NumChildren > 0 {
			continue
		}

		b, _, err := z.client.Get(nodePath(name, n))
		if err != nil {
			return nil, err
		}

		sn, err := decode(b)
		if err != nil {
			return nil, err
		}

		s, ok := serviceMap[sn.Version]
		if !ok {
			s = &registry.Service{
				Name:      sn.Name,
				Version:   sn.Version,
				Metadata:  sn.Metadata,
				Endpoints: sn.Endpoints,
			}
			serviceMap[s.Version] = s
		}

		for _, node := range sn.Nodes {
			s.Nodes = append(s.Nodes, node)
		}
	}

	services := make([]*registry.Service, 0, len(serviceMap))

	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (z *zookeeperRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	srv, _, err := z.client.Children(prefix)
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*registry.Service)

	for _, key := range srv {
		s := servicePath(key)
		nodes, _, err := z.client.Children(s)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			_, stat, err := z.client.Children(nodePath(key, node))
			if err != nil {
				return nil, err
			}

			if stat.NumChildren == 0 {
				b, _, err := z.client.Get(nodePath(key, node))
				if err != nil {
					return nil, err
				}
				i, err := decode(b)
				if err != nil {
					return nil, err
				}
				serviceMap[s] = &registry.Service{Name: i.Name}
			}
		}
	}

	var services []*registry.Service

	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (z *zookeeperRegistry) String() string {
	return "zookeeper"
}

func (z *zookeeperRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newZookeeperWatcher(z, opts...)
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	var options registry.Options
	for _, o := range opts {
		o(&options)
	}

	if options.Timeout == 0 {
		options.Timeout = 5
	}

	cAddrs := make([]string, 0, len(options.Addrs))
	for _, addr := range options.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}

	if len(cAddrs) == 0 {
		cAddrs = []string{"127.0.0.1:2181"}
	}

	// connect to zookeeper
	c, _, err := zk.Connect(cAddrs, time.Second*options.Timeout)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	// create our prefix path
	if err := createPath(prefix, []byte{}, c); err != nil {
		log.Error(err.Error())
		return nil
	}

	return &zookeeperRegistry{
		client:   c,
		options:  options,
		register: make(map[string]uint64),
	}
}
