// package kubernetes implements kubernetes micro runtime
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
func NewRuntime(opts ...runtime.Option) *kubernetes {
	options := runtime.Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// kubernetes client
	client := client.NewClientInCluster()

	return &kubernetes{
		options: options,
		closed:  make(chan bool),
		start:   make(chan *runtime.Service, 128),
		client:  client,
	}
}

// Registers a service
func (k *kubernetes) Create(s *runtime.Service, opts ...runtime.CreateOption) error {
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
func (k *kubernetes) run() {
	k.RLock()
	closed := k.closed
	k.RUnlock()

	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			// check running services
			// TODO: noop for now, but might have to check if
			// * deployments exist
			// * service is exposed
		case service := <-k.start:
			// TODO: following might have to be done
			// * create a deployment
			// * expose a service
			log.Debugf("Starting service: %s", service.Name)
		case <-closed:
			log.Debugf("Runtime stopped. Attempting to stop all services.")
			for name, service := range k.services {
				// TODO: handle this error
				if err := k.Delete(service); err != nil {
					log.Debugf("Runtime failed to stop service %s: %v", name, err)
				}
			}
			return
		}
	}
}

// poll polls for updates and updates services when new update has been detected
func (k *kubernetes) poll() {
	t := time.NewTicker(k.options.Poller.Tick())
	defer t.Stop()

	for {
		select {
		case <-k.closed:
			return
		case <-t.C:
			// poll remote endpoint for updates
			resp, err := k.options.Poller.Poll()
			if err != nil {
				log.Debugf("error polling for updates: %v", err)
				continue
			}
			// parse returned response to timestamp
			buildTime, err := time.Parse(time.RFC3339, resp.Image)
			if err != nil {
				log.Debugf("error parsing build time: %v", err)
				continue
			}

			k.Lock()
			for _, service := range k.services {
				if service.Version == "" {
					// TODO: figure this one out
					log.Debugf("Could not parse service build; unknown")
					continue
				}
				muBuild, err := time.Parse(time.RFC3339, service.Version)
				if err != nil {
					log.Debugf("Could not parse %s service build: %v", service.Name, err)
					continue
				}
				if buildTime.After(muBuild) {
					muService := &runtime.Service{
						Name:    service.Name,
						Source:  service.Source,
						Path:    service.Path,
						Exec:    service.Exec,
						Version: resp.Image,
					}
					if err := k.Update(muService); err != nil {
						log.Debugf("error updating service %s: %v", service.Name, err)
						continue
					}
					service.Version = resp.Image
				}
			}
			k.Unlock()
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

	go k.run()

	if k.options.Poller != nil {
		go k.poll()
	}

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
	}

	return nil
}
