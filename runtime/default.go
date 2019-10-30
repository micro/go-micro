package runtime

import (
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	packager "github.com/micro/go-micro/runtime/package"
	"github.com/micro/go-micro/runtime/process"
	proc "github.com/micro/go-micro/runtime/process/os"
	"github.com/micro/go-micro/util/log"
)

type service struct {
	sync.RWMutex

	running bool
	closed  chan bool
	err     error

	// output for logs
	output io.Writer

	// service to manage
	*Service
	// process creator
	Process *proc.Process
	// Exec
	Exec *process.Executable
	// process pid
	PID *process.PID
}

func newService(s *Service, c CreateOptions) *service {
	var exec string
	var args []string

	if len(s.Exec) > 0 {
		parts := strings.Split(s.Exec, " ")
		exec = parts[0]
		args = []string{}

		if len(parts) > 1 {
			args = parts[1:]
		}
	} else {
		// set command
		exec = c.Command[0]
		// set args
		if len(c.Command) > 1 {
			args = c.Command[1:]
		}
	}

	return &service{
		Service: s,
		Process: new(proc.Process),
		Exec: &process.Executable{
			Binary: &packager.Binary{
				Name: s.Name,
				Path: exec,
			},
			Env:  c.Env,
			Args: args,
		},
		closed: make(chan bool),
		output: c.Output,
	}
}

func (s *service) streamOutput() {
	go io.Copy(s.output, s.PID.Output)
	go io.Copy(s.output, s.PID.Error)
}

// Running returns true is the service is running
func (s *service) Running() bool {
	s.RLock()
	defer s.RUnlock()
	return s.running
}

// Start stars the service
func (s *service) Start() error {
	s.Lock()
	defer s.Unlock()

	if s.running {
		return nil
	}

	// reset
	s.err = nil
	s.closed = make(chan bool)

	// TODO: pull source & build binary
	log.Debugf("Runtime service %s forking new process\n", s.Service.Name)
	p, err := s.Process.Fork(s.Exec)
	if err != nil {
		return err
	}

	// set the pid
	s.PID = p
	// set to running
	s.running = true

	if s.output != nil {
		s.streamOutput()
	}

	// wait and watch
	go s.Wait()

	return nil
}

// Stop stops the service
func (s *service) Stop() error {
	s.Lock()
	defer s.Unlock()

	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
		s.running = false
		if s.PID == nil {
			return nil
		}
		return s.Process.Kill(s.PID)
	}
}

// Error returns the last error service has returned
func (s *service) Error() error {
	s.RLock()
	defer s.RUnlock()
	return s.err
}

// Wait waits for the service to finish running
func (s *service) Wait() {
	// wait for process to exit
	err := s.Process.Wait(s.PID)

	s.Lock()
	defer s.Unlock()

	// save the error
	if err != nil {
		s.err = err
	}

	// no longer running
	s.running = false
}

type runtime struct {
	sync.RWMutex
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
func (r *runtime) run() {
	r.RLock()
	closed := r.closed
	r.RUnlock()

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
			log.Debugf("Starting %s", service.Name)
			if err := service.Start(); err != nil {
				log.Debugf("Runtime error starting %s: %v", service.Name, err)
			}
		case <-closed:
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

// poll polls for updates and updates services when new update has been detected
func (r *runtime) poll() {
	t := time.NewTicker(r.options.Poller.Tick())
	defer t.Stop()

	for {
		select {
		case <-r.closed:
			return
		case <-t.C:
			// poll remote endpoint for updates
			resp, err := r.options.Poller.Poll()
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
			r.Lock()
			for name, service := range r.services {
				if service.Version == "" {
					// TODO: figure this one out
					log.Debugf("Could not parse service build; unknown")
					continue
				}
				muBuild, err := time.Parse(time.RFC3339, service.Version)
				if err != nil {
					log.Debugf("Could not parse %s service build: %v", name, err)
					continue
				}
				if buildTime.After(muBuild) {
					if err := r.Update(service.Service); err != nil {
						log.Debugf("error updating service %s: %v", name, err)
						continue
					}
					service.Version = resp.Image
				}
			}
			r.Unlock()
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

	go r.run()

	if r.options.Poller != nil {
		go r.poll()
	}

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
	}

	return nil
}
