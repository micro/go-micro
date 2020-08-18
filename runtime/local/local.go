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
	"time"

	"github.com/hpcloud/tail"
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/runtime"
	"github.com/micro/go-micro/v3/runtime/local/git"
)

// defaultNamespace to use if not provided as an option
const defaultNamespace = "default"

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
	// used to stop the runtime
	closed chan bool
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
		closed:     make(chan bool),
		start:      make(chan *service, 128),
		namespaces: make(map[string]map[string]*service),
	}
}

func (r *localRuntime) checkoutSourceIfNeeded(s *runtime.Service, secrets map[string]string) error {
	// Runtime service like config have no source.
	// Skip checkout in that case
	if len(s.Source) == 0 {
		return nil
	}

	// Incoming uploaded files have format lastfolder.tar.gz or
	// lastfolder.tar.gz/relative/path
	sourceParts := strings.Split(s.Source, "/")
	compressedFilepath := filepath.Join(SourceDir, sourceParts[0])
	uncompressPath := strings.ReplaceAll(compressedFilepath, ".tar.gz", "")
	if len(sourceParts) > 1 {
		uncompressPath = filepath.Join(SourceDir, strings.ReplaceAll(sourceParts[0], ".tar.gz", ""))
	}

	// check if the directory already exists
	if ex, _ := exists(compressedFilepath); ex {
		err := os.RemoveAll(uncompressPath)
		if err != nil {
			return err
		}
		err = os.MkdirAll(uncompressPath, 0777)
		if err != nil {
			return err
		}
		err = git.Uncompress(compressedFilepath, uncompressPath)
		if err != nil {
			return err
		}
		if len(sourceParts) > 1 {
			lastFolderPart := s.Name
			fullp := append([]string{uncompressPath}, sourceParts[1:]...)
			s.Source = filepath.Join(append(fullp, lastFolderPart)...)
		} else {
			s.Source = uncompressPath
		}
		return nil
	}

	source, err := git.ParseSourceLocal("", s.Source)
	if err != nil {
		return err
	}
	source.Ref = s.Version

	err = git.CheckoutSource(os.TempDir(), source, secrets)
	if err != nil {
		return err
	}
	s.Source = source.FullPath
	return nil
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

// run runs the runtime management loop
func (r *localRuntime) run(events <-chan runtime.Event) {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	// process event processes an incoming event
	processEvent := func(event runtime.Event, service *service, ns string) error {
		// get current vals
		r.RLock()
		name := service.Name
		updated := service.updated
		r.RUnlock()

		// only process if the timestamp is newer
		if !event.Timestamp.After(updated) {
			return nil
		}

		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime updating service %s in %v namespace", name, ns)
		}

		// this will cause a delete followed by created
		if err := r.Update(service.Service, runtime.UpdateNamespace(ns)); err != nil {
			return err
		}

		// update the local timestamp
		r.Lock()
		service.updated = updated
		r.Unlock()

		return nil
	}

	for {
		select {
		case <-t.C:
			// check running services
			r.RLock()
			for _, sevices := range r.namespaces {
				for _, service := range sevices {
					if !service.ShouldStart() {
						continue
					}

					// TODO: check service error
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("Runtime starting %s", service.Name)
					}
					if err := service.Start(); err != nil {
						if logger.V(logger.DebugLevel, logger.DefaultLogger) {
							logger.Debugf("Runtime error starting %s: %v", service.Name, err)
						}
					}
				}
			}
			r.RUnlock()
		case service := <-r.start:
			if !service.ShouldStart() {
				continue
			}
			// TODO: check service error
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime starting service %s", service.Name)
			}
			if err := service.Start(); err != nil {
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debugf("Runtime error starting service %s: %v", service.Name, err)
				}
			}
		case event := <-events:
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime received notification event: %v", event)
			}
			// NOTE: we only handle Update events for now
			switch event.Type {
			case runtime.Update:
				if event.Service != nil {
					ns := defaultNamespace
					if event.Options != nil && len(event.Options.Namespace) > 0 {
						ns = event.Options.Namespace
					}

					r.RLock()
					if _, ok := r.namespaces[ns]; !ok {
						if logger.V(logger.DebugLevel, logger.DefaultLogger) {
							logger.Debugf("Runtime unknown namespace: %s", ns)
						}
						r.RUnlock()
						continue
					}
					service, ok := r.namespaces[ns][fmt.Sprintf("%v:%v", event.Service.Name, event.Service.Version)]
					r.RUnlock()
					if !ok {
						logger.Debugf("Runtime unknown service: %s", event.Service)
					}

					if err := processEvent(event, service, ns); err != nil {
						if logger.V(logger.DebugLevel, logger.DefaultLogger) {
							logger.Debugf("Runtime error updating service %s: %v", event.Service, err)
						}
					}
					continue
				}

				r.RLock()
				namespaces := r.namespaces
				r.RUnlock()

				// if blank service was received we update all services
				for ns, services := range namespaces {
					for _, service := range services {
						if err := processEvent(event, service, ns); err != nil {
							if logger.V(logger.DebugLevel, logger.DefaultLogger) {
								logger.Debugf("Runtime error updating service %s: %v", service.Name, err)
							}
						}
					}
				}
			}
		case <-r.closed:
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime stopped")
			}
			return
		}
	}
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
func (r *localRuntime) Create(s *runtime.Service, opts ...runtime.CreateOption) error {
	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}
	err := r.checkoutSourceIfNeeded(s, options.Secrets)
	if err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()

	if len(options.Namespace) == 0 {
		options.Namespace = defaultNamespace
	}
	if len(options.Command) == 0 {
		options.Command = []string{"go"}
		options.Args = []string{"run", "."}
	}

	// pass secrets as env vars
	for key, value := range options.Secrets {
		options.Env = append(options.Env, fmt.Sprintf("%v=%v", key, value))
	}

	if _, ok := r.namespaces[options.Namespace]; !ok {
		r.namespaces[options.Namespace] = make(map[string]*service)
	}
	if _, ok := r.namespaces[options.Namespace][serviceKey(s)]; ok {
		return errors.New("service already running")
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
func (r *localRuntime) Logs(s *runtime.Service, options ...runtime.LogsOption) (runtime.Logs, error) {
	lopts := runtime.LogsOptions{}
	for _, o := range options {
		o(&lopts)
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
func (r *localRuntime) Update(s *runtime.Service, opts ...runtime.UpdateOption) error {
	var options runtime.UpdateOptions
	for _, o := range opts {
		o(&options)
	}
	err := r.checkoutSourceIfNeeded(s, options.Secrets)
	if err != nil {
		return err
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

	return service.Start()
}

// Delete removes the service from the runtime and stops it
func (r *localRuntime) Delete(s *runtime.Service, opts ...runtime.DeleteOption) error {
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
	r.closed = make(chan bool)

	var events <-chan runtime.Event
	if r.options.Scheduler != nil {
		var err error
		events, err = r.options.Scheduler.Notify()
		if err != nil {
			// TODO: should we bail here?
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime failed to start update notifier")
			}
		}
	}

	go r.run(events)

	return nil
}

// Stop stops the runtime
func (r *localRuntime) Stop() error {
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
		for _, services := range r.namespaces {
			for _, service := range services {
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debugf("Runtime stopping %s", service.Name)
				}
				service.Stop()
			}
		}

		// stop the scheduler
		if r.options.Scheduler != nil {
			return r.options.Scheduler.Close()
		}
	}

	return nil
}

// String implements stringer interface
func (r *localRuntime) String() string {
	return "local"
}
