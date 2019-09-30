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
	running  bool
	services map[string]*Status
}

func (m *monitor) Check(service string) error {
	status, err := m.check(service)
	if err != nil {
		return err
	}
	m.Lock()
	m.services[service] = status
	m.Unlock()

	if status.Code != StatusRunning {
		return errors.New(status.Info)
	}

	return nil
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
			// TODO: checks that are not just RPC based
			// TODO: better matching of the protocol
			// TODO: maybe everything has to be a go-micro service?
			if node.Metadata["server"] != m.client.String() {
				continue
			}
			// check the transport matches
			if node.Metadata["transport"] != m.client.Options().Transport.String() {
				continue
			}

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

func (m *monitor) reap() {
	services, err := m.registry.ListServices()
	if err != nil {
		return
	}

	serviceMap := make(map[string]bool)
	for _, service := range services {
		serviceMap[service.Name] = true
	}

	m.Lock()
	defer m.Unlock()

	// range over our watched services
	for service, _ := range m.services {
		// check if the service exists in the registry
		if !serviceMap[service] {
			// if not, delete it in our status map
			delete(m.services, service)
		}
	}
}

func (m *monitor) run() {
	// check the status every tick
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	// reap dead services
	t2 := time.NewTicker(time.Hour)
	defer t2.Stop()

	// list the known services
	services, _ := m.registry.ListServices()

	// create a check chan of same length
	check := make(chan string, len(services))

	// front-load the services to watch
	for _, service := range services {
		check <- service.Name
	}

	for {
		select {
		// exit if we're told to
		case <-m.exit:
			return
		// check a service when told to
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
		// on the tick interval get all services and issue a check
		case <-t.C:
			// create a list of services
			serviceMap := make(map[string]bool)

			m.RLock()
			for service, _ := range m.services {
				serviceMap[service] = true
			}
			m.RUnlock()

			go func() {
				// check the status of all watched services
				for service, _ := range serviceMap {
					select {
					case <-m.exit:
						return
					case check <- service:
					default:
						// barf if we block
					}
				}

				// list services
				services, _ := m.registry.ListServices()

				for _, service := range services {
					// start watching the service
					if ok := serviceMap[service.Name]; !ok {
						m.Watch(service.Name)
					}
				}
			}()
		case <-t2.C:
			// reap any dead/non-existent services
			m.reap()
		}
	}
}

func (m *monitor) Reap(service string) error {
	services, err := m.registry.GetService(service)
	if err != nil {
		return nil
	}
	m.Lock()
	defer m.Unlock()
	delete(m.services, service)
	for _, service := range services {
		m.registry.Deregister(service)
	}
	return nil
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

func (m *monitor) Run() error {
	m.Lock()
	defer m.Unlock()

	if m.running {
		return nil
	}

	// reset the exit channel
	m.exit = make(chan bool)
	// setup a new cache
	m.registry = cache.New(m.options.Registry)

	// start running
	go m.run()

	// set to running
	m.running = true

	return nil
}

func (m *monitor) Stop() error {
	m.Lock()
	defer m.Unlock()

	if !m.running {
		return nil
	}

	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
		for s, _ := range m.services {
			delete(m.services, s)
		}
		m.registry.Stop()
		m.running = false
		return nil
	}

	return nil
}

func newMonitor(opts ...Option) Monitor {
	options := Options{
		Client:   client.DefaultClient,
		Registry: registry.DefaultRegistry,
	}

	for _, o := range opts {
		o(&options)
	}

	return &monitor{
		options:  options,
		exit:     make(chan bool),
		client:   options.Client,
		registry: cache.New(options.Registry),
		services: make(map[string]*Status),
	}
}
