package runtime

import (
	"errors"
	"sync"
	"time"

	"github.com/micro/go-micro/util/log"
)

type runtime struct {
	sync.RWMutex
	// options configure runtime
	options Options
	// used to stop the runtime
	closed chan bool
	// used to start new services
	start chan *service
	// indicates if we're running
	running bool
	// the service map
	services map[string]*service
}

func newRuntime(opts ...Option) *runtime {
	// get default options
	options := Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return &runtime{
		options:  options,
		closed:   make(chan bool),
		start:    make(chan *service, 128),
		services: make(map[string]*service),
	}
}

// run runs the runtime management loop
func (r *runtime) run(events <-chan Event) {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			// check running services
			r.RLock()
			for _, service := range r.services {
				if service.Running() {
					continue
				}

				// TODO: check service error
				log.Debugf("Runtime starting %s", service.Name)
				if err := service.Start(); err != nil {
					log.Debugf("Runtime error starting %s: %v", service.Name, err)
				}
			}
			r.RUnlock()
		case service := <-r.start:
			if service.Running() {
				continue
			}
			// TODO: check service error
			log.Debugf("Runtime starting service %s", service.Name)
			if err := service.Start(); err != nil {
				log.Debugf("Runtime error starting service %s: %v", service.Name, err)
			}
		case event := <-events:
			log.Debugf("Runtime received notification event: %v", event)
			// NOTE: we only handle Update events for now
			switch event.Type {
			case Update:
				// parse returned response to timestamp
				buildTime, err := time.Parse(time.RFC3339, event.Version)
				if err != nil {
					log.Debugf("Runtime error parsing build time: %v", err)
					continue
				}
				r.Lock()
				for _, service := range r.services {
					muBuild, err := time.Parse(time.RFC3339, service.Version)
					if err != nil {
						log.Debugf("Runtime could not parse %s service build: %v", service.Name, err)
						continue
					}
					if buildTime.After(muBuild) {
						if err := r.Update(service.Service); err != nil {
							log.Debugf("Runtime error updating service %s: %v", service.Name, err)
							continue
						}
						service.Version = event.Version
					}
				}
				r.Unlock()
			}
		case <-r.closed:
			log.Debugf("Runtime stopped. Attempting to stop all services.")
			for name, service := range r.services {
				// TODO: handle this error
				if err := r.Delete(service.Service); err != nil {
					log.Debugf("Runtime failed to stop service %s: %v", name, err)
				}
			}
			return
		}
	}
}

// Create creates a new service which is then started by runtime
func (r *runtime) Create(s *Service, opts ...CreateOption) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.services[s.Name]; ok {
		return errors.New("service already registered")
	}

	var options CreateOptions
	for _, o := range opts {
		o(&options)
	}

	if len(s.Exec) == 0 && len(options.Command) == 0 {
		return errors.New("missing exec command")
	}

	// save service
	r.services[s.Name] = newService(s, options)

	// push into start queue
	r.start <- r.services[s.Name]

	return nil
}

// Delete removes the service from the runtime and stops it
func (r *runtime) Delete(s *Service) error {
	r.Lock()
	defer r.Unlock()

	if s, ok := r.services[s.Name]; ok {
		delete(r.services, s.Name)
		return s.Stop()
	}

	return nil
}

// Update attemps to update the service
func (r *runtime) Update(s *Service) error {
	// delete the service
	if err := r.Delete(s); err != nil {
		return err
	}

	// create new service
	return r.Create(s)
}

// List returns a slice of all services tracked by the runtime
func (r *runtime) List() ([]*Service, error) {
	var services []*Service
	r.RLock()
	defer r.RUnlock()

	for _, service := range r.services {
		services = append(services, service.Service)
	}

	return services, nil
}

// Start starts the runtime
func (r *runtime) Start() error {
	r.Lock()
	defer r.Unlock()

	// already running
	if r.running {
		return nil
	}

	// set running
	r.running = true
	r.closed = make(chan bool)

	var events <-chan Event
	if r.options.Notifier != nil {
		var err error
		events, err = r.options.Notifier.Notify()
		if err != nil {
			// TODO: should we bail here?
			log.Debugf("Runtime failed to start update notifier")
		}
	}

	go r.run(events)

	return nil
}

// Stop stops the runtime
func (r *runtime) Stop() error {
	r.Lock()
	defer r.Unlock()

	if !r.running {
		return nil
	}

	select {
	case <-r.closed:
		return nil
	default:
		close(r.closed)

		// set not running
		r.running = false

		// stop all the services
		for _, service := range r.services {
			log.Debugf("Runtime stopping %s", service.Name)
			service.Stop()
		}
		// stop the notifier too
		if r.options.Notifier != nil {
			return r.options.Notifier.Close()
		}
	}

	return nil
}

// String implements stringer interface
func (r *runtime) String() string {
	return "local"
}
