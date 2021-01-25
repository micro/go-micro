package runtime

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/runtime/local/git"
	"github.com/nxadm/tail"
)

// defaultNamespace to use if not provided as an option
const defaultNamespace = "default"

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
	// namespaces stores services grouped by namespace, e.g. namespaces["foo"]["go.micro.auth:latest"]
	// would return the latest version of go.micro.auth from the foo namespace
	namespaces map[string]map[string]*service
}

// NewRuntime creates new local runtime and returns it
func NewRuntime(opts ...Option) Runtime {
	// get default options
	options := Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// make the logs directory
	path := filepath.Join(os.TempDir(), "micro", "logs")
	_ = os.MkdirAll(path, 0755)

	return &runtime{
		options:    options,
		closed:     make(chan bool),
		start:      make(chan *service, 128),
		namespaces: make(map[string]map[string]*service),
	}
}

// @todo move this to runtime default
func (r *runtime) checkoutSourceIfNeeded(s *Service) error {
	// Runtime service like config have no source.
	// Skip checkout in that case
	if len(s.Source) == 0 {
		return nil
	}
	// @todo make this come from config
	cpath := filepath.Join(os.TempDir(), "micro", "uploads", s.Source)
	path := strings.ReplaceAll(cpath, ".tar.gz", "")
	if ex, _ := exists(cpath); ex {
		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
		err = os.MkdirAll(path, 0777)
		if err != nil {
			return err
		}
		err = uncompress(cpath, path)
		if err != nil {
			return err
		}
		s.Source = path
		return nil
	}
	source, err := git.ParseSourceLocal("", s.Source)
	if err != nil {
		return err
	}
	source.Ref = s.Version

	err = git.CheckoutSource(os.TempDir(), source)
	if err != nil {
		return err
	}
	s.Source = source.FullPath
	return nil
}

// modified version of: https://gist.github.com/mimoo/25fc9716e0f1353791f5908f94d6e726
func uncompress(src string, dst string) error {
	file, err := os.OpenFile(src, os.O_RDWR|os.O_CREATE, 0666)
	defer file.Close()
	if err != nil {
		return err
	}
	// ungzip
	zr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	// untar
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := header.Name

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q\n", target)
		}

		// add dst + re-format slashes according to system
		target = filepath.Join(dst, header.Name)
		// if no join is needed, replace with ToSlash:
		// target = filepath.ToSlash(header.Name)

		// check the type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it (with 0755 permission)
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				// @todo think about this:
				// if we don't nuke the folder, we might end up with files from
				// the previous decompress.
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it (with same permission)
		case tar.TypeReg:
			// the truncating is probably unnecessary due to the `RemoveAll` of folders
			// above
			fileToWrite, err := os.OpenFile(target, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			if _, err := io.Copy(fileToWrite, tr); err != nil {
				return err
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}
	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
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

	// process event processes an incoming event
	processEvent := func(event Event, service *service, ns string) error {
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
		if err := r.Update(service.Service, UpdateNamespace(ns)); err != nil {
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
			case Update:
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
	path := filepath.Join(os.TempDir(), "micro", "logs")
	return filepath.Join(path, fmt.Sprintf("%v.log", name))
}

func serviceKey(s *Service) string {
	return fmt.Sprintf("%v:%v", s.Name, s.Version)
}

// Create creates a new service which is then started by runtime
func (r *runtime) Create(s *Service, opts ...CreateOption) error {
	err := r.checkoutSourceIfNeeded(s)
	if err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()

	var options CreateOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Namespace) == 0 {
		options.Namespace = defaultNamespace
	}
	if len(options.Command) == 0 {
		options.Command = []string{"go"}
		options.Args = []string{"run", "."}
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
func (r *runtime) Logs(s *Service, options ...LogsOption) (LogStream, error) {
	lopts := LogsOptions{}
	for _, o := range options {
		o(&lopts)
	}
	ret := &logStream{
		service: s.Name,
		stream:  make(chan LogRecord),
		stop:    make(chan bool),
	}

	fpath := logFile(s.Name)
	if ex, err := exists(fpath); err != nil {
		return nil, err
	} else if !ex {
		return nil, fmt.Errorf("Log file %v does not exists", fpath)
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
				ret.stream <- LogRecord{Message: line.Text}
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
	stream  chan LogRecord
	sync.Mutex
	stop chan bool
	err  error
}

func (l *logStream) Chan() chan LogRecord {
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
func (r *runtime) Read(opts ...ReadOption) ([]*Service, error) {
	r.Lock()
	defer r.Unlock()

	gopts := ReadOptions{}
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
	var services []*Service

	if _, ok := r.namespaces[gopts.Namespace]; !ok {
		return make([]*Service, 0), nil
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
func (r *runtime) Update(s *Service, opts ...UpdateOption) error {
	var options UpdateOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Namespace) == 0 {
		options.Namespace = defaultNamespace
	}

	err := r.checkoutSourceIfNeeded(s)
	if err != nil {
		return err
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
func (r *runtime) Delete(s *Service, opts ...DeleteOption) error {
	r.Lock()
	defer r.Unlock()

	var options DeleteOptions
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
func (r *runtime) String() string {
	return "local"
}
