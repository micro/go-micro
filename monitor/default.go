package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/micro/go-micro/client"
	pb "github.com/micro/go-micro/debug/proto"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/cache"
)

type monitor struct {
	options Options

	exit     chan bool
	registry cache.Cache
	client   client.Client

	sync.RWMutex
	services map[string]*Status
}

// check provides binary running/failed status.
// In the event Debug.Health cannot be called on a service we reap the node.
func (m *monitor) check(service string) (*Status, error) {
	services, err := m.registry.GetService(service)
	if err != nil {
		return nil, err
	}

	// create debug client
	debug := pb.NewDebugService(service, m.client)

	var status *Status
	var gerr error

	// iterate through multiple versions of a service
	for _, service := range services {
		for _, node := range service.Nodes {
			rsp, err := debug.Health(
				context.Background(),
				// empty health request
				&pb.HealthRequest{},
				// call this specific node
				client.WithAddress(node.Address),
				// retry in the event of failure
				client.WithRetries(3),
			)
			if err != nil {
				// reap the dead node
				m.registry.Deregister(&registry.Service{
					Name:    service.Name,
					Version: service.Version,
					Nodes:   []*registry.Node{node},
				})

				// save the error
				gerr = err
				continue
			}

			// expecting ok response status
			if rsp.Status != "ok" {
				gerr = errors.New(rsp.Status)
				continue
			}

			// no error set status
			status = &Status{
				Code: StatusRunning,
				Info: "running",
			}
		}
	}

	// if we got the success case return it
	if status != nil {
		return status, nil
	}

	// if gerr is not nil return it
	if gerr != nil {
		return &Status{
			Code:  StatusFailed,
			Info:  "not running",
			Error: gerr.Error(),
		}, nil
	}

	// otherwise unknown status
	return &Status{
		Code: StatusUnknown,
		Info: "unknown status",
	}, nil
}

func (m *monitor) Status(service string) (Status, error) {
	m.RLock()
	defer m.RUnlock()
	if status, ok := m.services[service]; ok {
		return *status, nil
	}
	return Status{}, ErrNotWatching
}

func (m *monitor) Watch(service string) error {
	m.Lock()
	defer m.Unlock()

	// check if we're watching
	if _, ok := m.services[service]; ok {
		return nil
	}

	// get the status
	status, err := m.check(service)
	if err != nil {
		return err
	}

	// set the status
	m.services[service] = status
	return nil
}

func (m *monitor) Stop() error {
	m.Lock()
	defer m.RUnlock()

	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
		for s, _ := range m.services {
			delete(m.services, s)
		}
		m.registry.Stop()
		return nil
	}

	return nil
}

func (m *monitor) run() {
	// check the status every tick
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	check := make(chan string)

	for {
		select {
		case <-m.exit:
			return
		case service := <-check:
			// check the status
			status, err := m.check(service)
			if err != nil {
				status = &Status{
					Code: StatusUnknown,
					Info: "unknown status",
				}
			}

			// save the status
			m.Lock()
			m.services[service] = status
			m.Unlock()
		case <-t.C:
			// create a list of services
			var services []string
			m.RLock()
			for service, _ := range m.services {
				services = append(services, service)
			}
			m.RUnlock()

			// check the status of all watched services
			for _, service := range services {
				select {
				case <-m.exit:
					return
				case check <- service:
				}
			}
		}
	}
}

func newMonitor(opts ...Option) Monitor {
	options := Options{
		Client:   client.DefaultClient,
		Registry: registry.DefaultRegistry,
	}

	for _, o := range opts {
		o(&options)
	}

	m := &monitor{
		options:  options,
		client:   options.Client,
		registry: cache.New(options.Registry),
		services: make(map[string]*Status),
	}

	go m.run()
	return m
}
