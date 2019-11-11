package runtime

import (
	"errors"
	"fmt"
	"strconv"
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

// NewRuntime creates new local runtime and returns it
func NewRuntime(opts ...Option) Runtime {
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

// Init initializes runtime options
func (r *runtime) Init(opts ...Option) error {
	r.Lock()
	defer r.Unlock()

	for _, o := range opts {
		o(&r.options)
	}

	return nil
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
				updateTimeStamp, err := strconv.ParseInt(event.Version, 10, 64)
				if err != nil {
					log.Debugf("Runtime error parsing build time for %s: %v", event.Service, err)
					continue
				}
				buildTime := time.Unix(updateTimeStamp, 0)
				processEvent := func(event Event, service *Service) error {
					buildTimeStamp, err := strconv.ParseInt(service.Version, 10, 64)
					if err != nil {
						return err
					}
					muBuild := time.Unix(buildTimeStamp, 0)
					if buildTime.After(muBuild) {
						if err := r.Update(service); err != nil {
							return err
						}
						service.Version = fmt.Sprintf("%d", buildTime.Unix())
					}
					return nil
				}
				r.Lock()
				if len(event.Service) > 0 {
					service, ok := r.services[event.Service]
					if !ok {
						log.Debugf("Runtime unknown service: %s", event.Service)
						r.Unlock()
						continue
					}
					if err := processEvent(event, service.Service); err != nil {
						log.Debugf("Runtime error updating service %s: %v", event.Service, err)
					}
					r.Unlock()
					continue
				}
				// if blank service was received we update all services
				for _, service := range r.services {
					if err := processEvent(event, service.Service); err != nil {
						log.Debugf("Runtime error updating service %s: %v", service.Name, err)
					}
				}
				r.Unlock()
			}
		case <-r.closed:
			log.Debugf("Runtime stopped.")
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
