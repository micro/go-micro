package runtime

import (
	"errors"
	"os"
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
	closed   chan bool
	running  bool
	services map[string]*service
}

type service struct {
	sync.RWMutex

	running bool
	closed  chan bool
	err     error

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
		services: make(map[string]*service),
	}
}

func newService(s *Service) *service {
	parts := strings.Split(s.Exec, " ")
	exec := parts[0]
	args := []string{}

	if len(parts) > 1 {
		args = parts[1:]
	}

	return &service{
		Service: s,
		Process: new(proc.Process),
		Exec: &process.Executable{
			Binary: &packager.Binary{
				Name: s.Name,
				Path: exec,
			},
			Env:  os.Environ(),
			Args: args,
		},
	}
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

	p, err := s.Process.Fork(s.Exec)
	if err != nil {
		return err
	}

	// set the pid
	s.PID = p
	// set to running
	s.running = true

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

func (r *runtime) Register(s *Service) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.services[s.Name]; ok {
		return errors.New("service already registered")
	}

	// save service
	r.services[s.Name] = newService(s)

	return nil
}

func (r *runtime) Run() error {
	r.Lock()

	// already running
	if r.running {
		r.Unlock()
		return nil
	}

	// set running
	r.running = true
	r.closed = make(chan bool)
	closed := r.closed

	r.Unlock()

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
				log.Debugf("Starting %s", service.Name)
				if err := service.Start(); err != nil {
					log.Debugf("Error starting %s: %v", service.Name, err)
				}
			}
			r.RUnlock()
		case <-closed:
			// TODO: stop all the things
			return nil
		}
	}

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
			service.Stop()
		}
	}

	return nil
}
