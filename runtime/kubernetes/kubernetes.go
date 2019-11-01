// Package kubernetes implements kubernetes micro runtime
package kubernetes

import (
	"errors"
	"sync"
	"time"

	"github.com/micro/go-micro/runtime"
	"github.com/micro/go-micro/runtime/kubernetes/client"
	"github.com/micro/go-micro/util/log"
)

type kubernetes struct {
	sync.RWMutex
	// options configure runtime
	options runtime.Options
	// indicates if we're running
	running bool
	// used to start new services
	start chan *runtime.Service
	// used to stop the runtime
	closed chan bool
	// service tracks deployed services
	services map[string]*runtime.Service
	// client is kubernetes client
	client client.Kubernetes
}

// NewRuntime creates new kubernetes runtime
func NewRuntime(opts ...runtime.Option) runtime.Runtime {
	// get default options
	options := runtime.Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// kubernetes client
	client := client.NewClientInCluster()

	return &kubernetes{
		options:  options,
		closed:   make(chan bool),
		start:    make(chan *runtime.Service, 128),
		services: make(map[string]*runtime.Service),
		client:   client,
	}
}

// Init initializes runtime options
func (k *kubernetes) Init(opts ...runtime.Option) error {
	k.Lock()
	defer k.Unlock()

	for _, o := range opts {
		o(&k.options)
	}

	return nil
}

// Registers a service
func (k *kubernetes) Create(s *runtime.Service, opts ...runtime.CreateOption) error {
	k.Lock()
	defer k.Unlock()

	// TODO:
	// * create service
	// * create deployment

	// NOTE: our services have micro- prefix
	s.Name = "micro-" + s.Name

	// NOTE: we are tracking this in memory for now
	if _, ok := k.services[s.Name]; ok {
		return errors.New("service already registered")
	}

	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}

	// save service
	k.services[s.Name] = s
	// push into start queue
	k.start <- k.services[s.Name]

	return nil
}

// Remove a service
func (k *kubernetes) Delete(s *runtime.Service) error {
	k.Lock()
	defer k.Unlock()

	// TODO:
	// * delete service
	// * delete dpeloyment

	// NOTE: we are tracking this in memory for now
	if s, ok := k.services[s.Name]; ok {
		delete(k.services, s.Name)
		return nil
	}

	return nil
}

// Update the service in place
func (k *kubernetes) Update(s *runtime.Service) error {
	// metada which we will PATCH deployment with
	metadata := &client.Metadata{
		Annotations: map[string]string{
			"build": s.Version,
		},
	}
	return k.client.UpdateDeployment(s.Name, metadata)
}

// List the managed services
func (k *kubernetes) List() ([]*runtime.Service, error) {
	// TODO: this should list the k8s deployments
	// but for now we return in-memory tracked services
	var services []*runtime.Service
	k.RLock()
	defer k.RUnlock()

	for _, service := range k.services {
		services = append(services, service)
	}

	return services, nil
}

// run runs the runtime management loop
func (k *kubernetes) run(events <-chan runtime.Event) {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			// TODO: noop for now
			// check running services
			// * deployments exist
			// * service is exposed
		case service := <-k.start:
			// TODO: following might have to be done
			// * create a deployment
			// * expose a service
			log.Debugf("Runtime starting service: %s", service.Name)
		case event := <-events:
			// NOTE: we only handle Update events for now
			log.Debugf("Runtime received notification event: %v", event)
			switch event.Type {
			case runtime.Update:
				// parse returned response to timestamp
				buildTime, err := time.Parse(time.RFC3339, event.Version)
				if err != nil {
					log.Debugf("Runtime error parsing build time: %v", err)
					continue
				}
				processEvent := func(event runtime.Event, service *runtime.Service) error {
					muBuild, err := time.Parse(time.RFC3339, service.Version)
					if err != nil {
						return err
					}
					if buildTime.After(muBuild) {
						muService := &runtime.Service{
							Name:    service.Name,
							Source:  service.Source,
							Path:    service.Path,
							Exec:    service.Exec,
							Version: event.Version,
						}
						if err := k.Update(muService); err != nil {
							return err
						}
						service.Version = event.Version
					}
					return nil
				}
				k.Lock()
				if len(event.Service) > 0 {
					service, ok := k.services[event.Service]
					if !ok {
						log.Debugf("Runtime unknown service: %s", event.Service)
						k.Unlock()
						continue
					}
					if err := processEvent(event, service); err != nil {
						log.Debugf("Runtime error updating service %s: %v", event.Service, err)
					}
					k.Unlock()
					continue
				}
				// if blank service was received we update all services
				for _, service := range k.services {
					if err := processEvent(event, service); err != nil {
						log.Debugf("Runtime error updating service %s: %v", event.Service, err)
					}
				}
				k.Unlock()
			}
		case <-k.closed:
			log.Debugf("Runtime stopped")
			return
		}
	}
}

// starts the runtime
func (k *kubernetes) Start() error {
	k.Lock()
	defer k.Unlock()

	// already running
	if k.running {
		return nil
	}

	// set running
	k.running = true
	k.closed = make(chan bool)

	var events <-chan runtime.Event
	if k.options.Notifier != nil {
		var err error
		events, err = k.options.Notifier.Notify()
		if err != nil {
			// TODO: should we bail here?
			log.Debugf("Runtime failed to start update notifier")
		}
	}

	go k.run(events)

	return nil
}

// Shutdown the runtime
func (k *kubernetes) Stop() error {
	k.Lock()
	defer k.Unlock()

	if !k.running {
		return nil
	}

	select {
	case <-k.closed:
		return nil
	default:
		close(k.closed)
		// set not running
		k.running = false
		// stop the notifier too
		if k.options.Notifier != nil {
			return k.options.Notifier.Close()
		}
	}

	return nil
}

// String implements stringer interface
func (k *kubernetes) String() string {
	return "kubernetes"
}
