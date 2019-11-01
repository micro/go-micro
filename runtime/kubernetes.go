// package kubernetes implements kubernetes micro runtime
package runtime

import (
	"errors"
	"sync"
	"time"

	"github.com/micro/go-micro/runtime/kubernetes/client"
	"github.com/micro/go-micro/util/log"
)

type kubernetes struct {
	sync.RWMutex
	// options configure runtime
	options Options
	// indicates if we're running
	running bool
	// used to start new services
	start chan *Service
	// used to stop the runtime
	closed chan bool
	// service tracks deployed services
	services map[string]*Service
	// client is kubernetes client
	client client.Kubernetes
}

// NewK8sRuntime creates new kubernetes runtime
func NewK8sRuntime(opts ...Option) Runtime {
	// get default options
	options := Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// kubernetes client
	client := client.NewClientInCluster()

	return &kubernetes{
		options: options,
		closed:  make(chan bool),
		start:   make(chan *Service, 128),
		client:  client,
	}
}

// Init initializes runtime options
func (k *kubernetes) Init(opts ...Option) error {
	k.Lock()
	defer k.Unlock()

	for _, o := range opts {
		o(&k.options)
	}

	return nil
}

// Registers a service
func (k *kubernetes) Create(s *Service, opts ...CreateOption) error {
	k.Lock()
	defer k.Unlock()

	// TODO:
	// * create service
	// * create dpeloyment

	// NOTE: our services have micro- prefix
	s.Name = "micro-" + s.Name

	// NOTE: we are tracking this in memory for now
	if _, ok := k.services[s.Name]; ok {
		return errors.New("service already registered")
	}

	var options CreateOptions
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
func (k *kubernetes) Delete(s *Service) error {
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
func (k *kubernetes) Update(s *Service) error {
	// metada which we will PATCH deployment with
	metadata := &client.Metadata{
		Annotations: map[string]string{
			"build": s.Version,
		},
	}
	return k.client.UpdateDeployment(s.Name, metadata)
}

// List the managed services
func (k *kubernetes) List() ([]*Service, error) {
	// TODO: this should list the k8s deployments
	// but for now we return in-memory tracked services
	var services []*Service
	k.RLock()
	defer k.RUnlock()

	for _, service := range k.services {
		services = append(services, service)
	}

	return services, nil
}

// run runs the runtime management loop
func (k *kubernetes) run(events <-chan Event) {
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
			case Update:
				// parse returned response to timestamp
				buildTime, err := time.Parse(time.RFC3339, event.Version)
				if err != nil {
					log.Debugf("Runtime error parsing build time: %v", err)
					continue
				}
				k.Lock()
				for _, service := range k.services {
					muBuild, err := time.Parse(time.RFC3339, service.Version)
					if err != nil {
						log.Debugf("Runtime could not parse %s service build: %v", service.Name, err)
						continue
					}
					if buildTime.After(muBuild) {
						muService := &Service{
							Name:    service.Name,
							Source:  service.Source,
							Path:    service.Path,
							Exec:    service.Exec,
							Version: event.Version,
						}
						if err := k.Update(muService); err != nil {
							log.Debugf("Runtime error updating service %s: %v", service.Name, err)
							continue
						}
						service.Version = event.Version
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

	var events <-chan Event
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
