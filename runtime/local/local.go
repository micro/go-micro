package local

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hpcloud/tail"
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/runtime"
)

// defaultNamespace to use if not provided as an option
const defaultNamespace = "micro"

var (
	// The directory for logs to be output
	LogDir = filepath.Join(os.TempDir(), "micro", "logs")
	// The source directory where code lives
	SourceDir = filepath.Join(os.TempDir(), "micro", "uploads")
)

type localRuntime struct {
	sync.RWMutex
	// options configure runtime
	options runtime.Options
	// used to start new services
	start chan *service
	// indicates if we're running
	running bool
	// namespaces stores services grouped by namespace, e.g. namespaces["foo"]["go.micro.auth:latest"]
	// would return the latest version of go.micro.auth from the foo namespace
	namespaces map[string]map[string]*service
}

// NewRuntime creates new local runtime and returns it
func NewRuntime(opts ...runtime.Option) runtime.Runtime {
	// get default options
	options := runtime.Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// make the logs directory
	os.MkdirAll(LogDir, 0755)

	return &localRuntime{
		options:    options,
		start:      make(chan *service, 128),
		namespaces: make(map[string]map[string]*service),
	}
}

// Init initializes runtime options
func (r *localRuntime) Init(opts ...runtime.Option) error {
	r.Lock()
	defer r.Unlock()

	for _, o := range opts {
		o(&r.options)
	}

	return nil
}

func logFile(serviceName string) string {
	// make the directory
	name := strings.Replace(serviceName, "/", "-", -1)
	return filepath.Join(LogDir, fmt.Sprintf("%v.log", name))
}

func serviceKey(s *runtime.Service) string {
	return fmt.Sprintf("%v:%v", s.Name, s.Version)
}

// Create creates a new service which is then started by runtime
func (r *localRuntime) Create(resource runtime.Resource, opts ...runtime.CreateOption) error {
	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}

	r.Lock()
	defer r.Unlock()

	// Handle the various different types of resources:
	switch resource.Type() {
	case runtime.TypeNamespace:
		// noop (Namespace is not supported by local)
		return nil
	case runtime.TypeNetworkPolicy:
		// noop (NetworkPolicy is not supported by local)
		return nil
	case runtime.TypeService:

		// Assert the resource back into a *runtime.Service
		s, ok := resource.(*runtime.Service)
		if !ok {
			return runtime.ErrInvalidResource
		}

		if len(options.Namespace) == 0 {
			options.Namespace = defaultNamespace
		}
		if len(options.Entrypoint) > 0 {
			s.Source = filepath.Join(s.Source, options.Entrypoint)
		}
		if len(options.Command) == 0 {
			ep, err := Entrypoint(s.Source)
			if err != nil {
				return err
			}

			options.Command = []string{"go"}
			options.Args = []string{"run", ep}
		}

		// pass secrets as env vars
		for key, value := range options.Secrets {
			options.Env = append(options.Env, fmt.Sprintf("%v=%v", key, value))
		}

		if _, ok := r.namespaces[options.Namespace]; !ok {
			r.namespaces[options.Namespace] = make(map[string]*service)
		}
		if _, ok := r.namespaces[options.Namespace][serviceKey(s)]; ok {
			return runtime.ErrAlreadyExists
		}

		// create new service
		service := newService(s, options)

		f, err := os.OpenFile(logFile(service.Name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		if service.output != nil {
			service.output = io.MultiWriter(service.output, f)
		} else {
			service.output = f
		}
		// start the service
		if err := service.Start(); err != nil {
			return err
		}
		// save service
		r.namespaces[options.Namespace][serviceKey(s)] = service

		return nil
	default:
		return runtime.ErrInvalidResource
	}
}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// @todo: Getting existing lines is not supported yet.
// The reason for this is because it's hard to calculate line offset
// as opposed to character offset.
// This logger streams by default and only supports the `StreamCount` option.
func (r *localRuntime) Logs(resource runtime.Resource, options ...runtime.LogsOption) (runtime.Logs, error) {
	lopts := runtime.LogsOptions{}
	for _, o := range options {
		o(&lopts)
	}

	// Handle the various different types of resources:
	switch resource.Type() {
	case runtime.TypeNamespace:
		// noop (Namespace is not supported by local)
		return nil, nil
	case runtime.TypeNetworkPolicy:
		// noop (NetworkPolicy is not supported by local)
		return nil, nil
	case runtime.TypeService:

		// Assert the resource back into a *runtime.Service
		s, ok := resource.(*runtime.Service)
		if !ok {
			return nil, runtime.ErrInvalidResource
		}

		ret := &logStream{
			service: s.Name,
			stream:  make(chan runtime.Log),
			stop:    make(chan bool),
		}

		fpath := logFile(s.Name)
		if ex, err := exists(fpath); err != nil {
			return nil, err
		} else if !ex {
			return nil, fmt.Errorf("Logs not found for service %s", s.Name)
		}

		// have to check file size to avoid too big of a seek
		fi, err := os.Stat(fpath)
		if err != nil {
			return nil, err
		}
		size := fi.Size()

		whence := 2
		// Multiply by length of an average line of log in bytes
		offset := lopts.Count * 200

		if offset > size {
			offset = size
		}
		offset *= -1

		t, err := tail.TailFile(fpath, tail.Config{Follow: lopts.Stream, Location: &tail.SeekInfo{
			Whence: whence,
			Offset: int64(offset),
		}, Logger: tail.DiscardingLogger})
		if err != nil {
			return nil, err
		}

		ret.tail = t
		go func() {
			for {
				select {
				case line, ok := <-t.Lines:
					if !ok {
						ret.Stop()
						return
					}
					ret.stream <- runtime.Log{Message: line.Text}
				case <-ret.stop:
					return
				}
			}

		}()
		return ret, nil
	default:
		return nil, runtime.ErrInvalidResource
	}
}

type logStream struct {
	tail    *tail.Tail
	service string
	stream  chan runtime.Log
	sync.Mutex
	stop chan bool
	err  error
}

func (l *logStream) Chan() chan runtime.Log {
	return l.stream
}

func (l *logStream) Error() error {
	return l.err
}

func (l *logStream) Stop() error {
	l.Lock()
	defer l.Unlock()

	select {
	case <-l.stop:
		return nil
	default:
		close(l.stop)
		close(l.stream)
		err := l.tail.Stop()
		if err != nil {
			logger.Errorf("Error stopping tail: %v", err)
			return err
		}
	}
	return nil
}

// Read returns all instances of requested service
// If no service name is provided we return all the track services.
func (r *localRuntime) Read(opts ...runtime.ReadOption) ([]*runtime.Service, error) {
	r.Lock()
	defer r.Unlock()

	gopts := runtime.ReadOptions{}
	for _, o := range opts {
		o(&gopts)
	}
	if len(gopts.Namespace) == 0 {
		gopts.Namespace = defaultNamespace
	}

	save := func(k, v string) bool {
		if len(k) == 0 {
			return true
		}
		return k == v
	}

	//nolint:prealloc
	var services []*runtime.Service

	if _, ok := r.namespaces[gopts.Namespace]; !ok {
		return make([]*runtime.Service, 0), nil
	}

	for _, service := range r.namespaces[gopts.Namespace] {
		if !save(gopts.Service, service.Name) {
			continue
		}
		if !save(gopts.Version, service.Version) {
			continue
		}
		// TODO deal with service type
		// no version has sbeen requested, just append the service
		services = append(services, service.Service)
	}

	return services, nil
}

// Update attempts to update the service
func (r *localRuntime) Update(resource runtime.Resource, opts ...runtime.UpdateOption) error {
	var options runtime.UpdateOptions
	for _, o := range opts {
		o(&options)
	}

	// Handle the various different types of resources:
	switch resource.Type() {
	case runtime.TypeNamespace:
		// noop (Namespace is not supported by local)
		return nil
	case runtime.TypeNetworkPolicy:
		// noop (NetworkPolicy is not supported by local)
		return nil
	case runtime.TypeService:

		// Assert the resource back into a *runtime.Service
		s, ok := resource.(*runtime.Service)
		if !ok {
			return runtime.ErrInvalidResource
		}

		if len(options.Entrypoint) > 0 {
			s.Source = filepath.Join(s.Source, options.Entrypoint)
		}

		if len(options.Namespace) == 0 {
			options.Namespace = defaultNamespace
		}

		r.Lock()
		srvs, ok := r.namespaces[options.Namespace]
		r.Unlock()
		if !ok {
			return errors.New("Service not found")
		}

		r.Lock()
		service, ok := srvs[serviceKey(s)]
		r.Unlock()
		if !ok {
			return errors.New("Service not found")
		}

		if err := service.Stop(); err != nil && err.Error() != "no such process" {
			logger.Errorf("Error stopping service %s: %s", service.Name, err)
			return err
		}

		// update the source to the new location and restart the service
		service.Source = s.Source
		service.Exec.Dir = s.Source
		return service.Start()

	default:
		return runtime.ErrInvalidResource
	}
}

// Delete removes the service from the runtime and stops it
func (r *localRuntime) Delete(resource runtime.Resource, opts ...runtime.DeleteOption) error {

	// Handle the various different types of resources:
	switch resource.Type() {
	case runtime.TypeNamespace:
		// noop (Namespace is not supported by local)
		return nil
	case runtime.TypeNetworkPolicy:
		// noop (NetworkPolicy is not supported by local)
		return nil
	case runtime.TypeService:

		// Assert the resource back into a *runtime.Service
		s, ok := resource.(*runtime.Service)
		if !ok {
			return runtime.ErrInvalidResource
		}

		r.Lock()
		defer r.Unlock()

		var options runtime.DeleteOptions
		for _, o := range opts {
			o(&options)
		}
		if len(options.Namespace) == 0 {
			options.Namespace = defaultNamespace
		}

		srvs, ok := r.namespaces[options.Namespace]
		if !ok {
			return nil
		}

		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime deleting service %s", s.Name)
		}

		service, ok := srvs[serviceKey(s)]
		if !ok {
			return nil
		}

		// check if running
		if !service.Running() {
			delete(srvs, service.key())
			r.namespaces[options.Namespace] = srvs
			return nil
		}
		// otherwise stop it
		if err := service.Stop(); err != nil {
			return err
		}
		// delete it
		delete(srvs, service.key())
		r.namespaces[options.Namespace] = srvs
		return nil
	default:
		return runtime.ErrInvalidResource
	}
}

// Start starts the runtime
func (r *localRuntime) Start() error {
	r.Lock()
	defer r.Unlock()

	// already running
	if r.running {
		return nil
	}

	// set running
	r.running = true
	return nil
}

// Stop stops the runtime
func (r *localRuntime) Stop() error {
	r.Lock()
	defer r.Unlock()

	if !r.running {
		return nil
	}

	// set not running
	r.running = false

	// stop all the services
	for _, services := range r.namespaces {
		for _, service := range services {
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime stopping %s", service.Name)
			}
			service.Stop()
		}
	}

	return nil
}

// String implements stringer interface
func (r *localRuntime) String() string {
	return "local"
}

// Entrypoint determines the entrypoint for the service, since main.go doesn't always exist at
// the top level
func Entrypoint(dir string) (string, error) {
	var entrypoints []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// get the relative path to the directory
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// only look for files in the top level or the cmd folder
		if dir := filepath.Dir(rel); !filepath.HasPrefix(dir, "cmd") && dir != "." {
			return nil
		}

		// only look for main.go files
		if filepath.Base(rel) == "main.go" {
			entrypoints = append(entrypoints, rel)
		}

		return nil
	})

	switch len(entrypoints) {
	case 0:
		return "", errors.New("No entrypoint found")
	case 1:
		return entrypoints[0], nil
	default:
		return "", errors.New("More than one entrypoint found")
	}
}
