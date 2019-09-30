package runtime

import (
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/runtime/package"
	"github.com/micro/go-micro/runtime/process"
	proc "github.com/micro/go-micro/runtime/process/os"
	"github.com/micro/go-micro/util/log"
)

type runtime struct {
	sync.RWMutex
	// used to stop the runtime
	closed chan bool
	// used to start new services
	start chan *service
	// indicates if we're running
	running bool
	// the service map
	services map[string]*service
}

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

func newRuntime() *runtime {
	return &runtime{
		closed:   make(chan bool),
		start:    make(chan *service, 128),
		services: make(map[string]*service),
	}
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
		output: c.Output,
	}
}

func (s *service) streamOutput() {
	go io.Copy(s.output, s.PID.Output)
	go io.Copy(s.output, s.PID.Error)
}

func (s *service) Running() bool {
	s.RLock()
	defer s.RUnlock()
	return s.running
}

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
	log.Debugf("Runtime service %s forking new process\n")
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

func (s *service) Stop() error {
	s.Lock()
	defer s.Unlock()

	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
		s.running = false
		return s.Process.Kill(s.PID)
	}

	return nil
}

func (s *service) Error() error {
	s.RLock()
	defer s.RUnlock()
	return s.err
}

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
			// TODO: stop all the things
			return
		}
	}
}

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

func (r *runtime) Delete(s *Service) error {
	r.Lock()
	defer r.Unlock()

	if s, ok := r.services[s.Name]; ok {
		delete(r.services, s.Name)
		return s.Stop()
	}

	return nil
}

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

	return nil
}

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
