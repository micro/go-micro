// Package kubernetes implements kubernetes micro runtime
package kubernetes

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/runtime"
	"github.com/micro/go-micro/runtime/kubernetes/client"
	"github.com/micro/go-micro/util/log"
)

// action to take on runtime service
type action int

const (
	start action = iota
	update
	stop
)

// task is queued into runtime queue
type task struct {
	action  action
	service *service
}

type kubernetes struct {
	sync.RWMutex
	// options configure runtime
	options runtime.Options
	// indicates if we're running
	running bool
	// task queue for kubernetes services
	queue chan *task
	// used to stop the runtime
	closed chan bool
	// client is kubernetes client
	client client.Kubernetes
}

// NewRuntime creates new kubernetes runtime
func NewRuntime(opts ...runtime.Option) runtime.Runtime {
	// get default options
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
		queue:   make(chan *task, 128),
		client:  client,
	}
}

// Init initializes runtime options
func (k *kubernetes) Init(opts ...runtime.Option) error {
	k.Lock()
	defer k.Unlock()

	for _, o := range opts {
		o(&k.options)
	}

	return nil
}

// Creates a service
func (k *kubernetes) Create(s *runtime.Service, opts ...runtime.CreateOption) error {
	k.Lock()
	defer k.Unlock()

	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}

	svcName := s.Name
	if len(s.Version) > 0 {
		svcName = strings.Join([]string{s.Name, s.Version}, "-")
	}

	if !client.ServiceRegexp.MatchString(svcName) {
		return fmt.Errorf("invalid service name: %s", svcName)
	}

	// create new kubernetes micro service
	service := newService(s, options)

	log.Debugf("Runtime queueing service %s for start action", service.Name)

	// push into start queue
	k.queue <- &task{
		action:  start,
		service: service,
	}

	return nil
}

// Get returns all instances of given service
func (k *kubernetes) Get(name string, opts ...runtime.GetOption) ([]*runtime.Service, error) {
	k.Lock()
	defer k.Unlock()

	// if no name has been passed in, return error
	if len(name) == 0 {
		return nil, errors.New("missing service name")
	}

	// set the default label
	labels := map[string]string{
		"micro": "service",
		"name":  name,
	}
	var options runtime.GetOptions
	for _, o := range opts {
		o(&options)
	}

	// add version to labels if a version has been supplied
	if len(options.Version) > 0 {
		labels["version"] = options.Version
	}

	log.Debugf("Runtime querying service %s", name)

	serviceList := new(client.ServiceList)
	r := &client.Resource{
		Kind:  "service",
		Value: serviceList,
	}
	if err := k.client.Get(r, labels); err != nil {
		return nil, err
	}

	services := make([]*runtime.Service, 0, len(serviceList.Items))
	for _, kservice := range serviceList.Items {
		service := &runtime.Service{
			Name:    kservice.Metadata.Name,
			Version: kservice.Metadata.Version,
		}
		services = append(services, service)
	}

	return services, nil
}

// Update the service in place
func (k *kubernetes) Update(s *runtime.Service) error {
	// parse version into human readable timestamp
	updateTimeStamp, err := strconv.ParseInt(s.Version, 10, 64)
	if err != nil {
		return err
	}
	unixTimeUTC := time.Unix(updateTimeStamp, 0)

	// create new kubernetes micro service
	service := newService(s, runtime.CreateOptions{})

	// update build time annotation
	service.kdeploy.Spec.Template.Metadata.Annotations["build"] = unixTimeUTC.Format(time.RFC3339)

	log.Debugf("Runtime queueing service %s for update action", service.Name)

	// queue service for removal
	k.queue <- &task{
		action:  update,
		service: service,
	}

	return nil
}

// Remove a service
func (k *kubernetes) Delete(s *runtime.Service) error {
	k.Lock()
	defer k.Unlock()

	// create new kubernetes micro service
	service := newService(s, runtime.CreateOptions{})

	log.Debugf("Runtime queueing service %s for delete action", service.Name)

	// queue service for removal
	k.queue <- &task{
		action:  stop,
		service: service,
	}

	return nil
}

// List the managed services
func (k *kubernetes) List() ([]*runtime.Service, error) {
	serviceList := new(client.ServiceList)
	r := &client.Resource{
		Kind:  "service",
		Value: serviceList,
	}

	if err := k.client.List(r); err != nil {
		return nil, err
	}

	log.Debugf("Runtime found %d micro services", len(serviceList.Items))

	services := make([]*runtime.Service, 0, len(serviceList.Items))

	for _, service := range serviceList.Items {
		buildTime, err := time.Parse(time.RFC3339, service.Metadata.Annotations["build"])
		if err != nil {
			log.Debugf("Runtime error parsing build time for %s: %v", service.Metadata.Name, err)
			continue
		}
		// add the service to the list of services
		svc := &runtime.Service{
			Name:    service.Metadata.Name,
			Version: fmt.Sprintf("%d", buildTime.Unix()),
		}
		services = append(services, svc)
	}

	return services, nil
}

// run runs the runtime management loop
func (k *kubernetes) run(events <-chan runtime.Event) {
	t := time.NewTicker(time.Second * 10)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			// TODO: figure out what to do here
			// - do we even need the ticker for k8s services?
		case task := <-k.queue:
			switch task.action {
			case start:
				log.Debugf("Runtime starting new service: %s", task.service.Name)
				if err := task.service.Start(k.client); err != nil {
					log.Debugf("Runtime failed to start service %s: %v", task.service.Name, err)
					continue
				}
			case stop:
				log.Debugf("Runtime stopping service: %s", task.service.Name)
				if err := task.service.Stop(k.client); err != nil {
					log.Debugf("Runtime failed to stop service %s: %v", task.service.Name, err)
					continue
				}
			case update:
				log.Debugf("Runtime updating service: %s", task.service.Name)
				if err := task.service.Update(k.client); err != nil {
					log.Debugf("Runtime failed to update service %s: %v", task.service.Name, err)
					continue
				}
			default:
				log.Debugf("Runtime received unknown action for service: %s", task.service.Name)
			}
		case event := <-events:
			// NOTE: we only handle Update events for now
			log.Debugf("Runtime received notification event: %v", event)
			switch event.Type {
			case runtime.Update:
				// parse returned response to timestamp
				updateTimeStamp, err := strconv.ParseInt(event.Version, 10, 64)
				if err != nil {
					log.Debugf("Runtime error parsing update build time: %v", err)
					continue
				}
				unixTimeUTC := time.Unix(updateTimeStamp, 0)
				if len(event.Service) > 0 {
					s := &runtime.Service{
						Name:    event.Service,
						Version: event.Version,
					}
					// create new kubernetes micro service
					service := newService(s, runtime.CreateOptions{})
					// update build time annotation
					service.kdeploy.Spec.Template.Metadata.Annotations["build"] = unixTimeUTC.Format(time.RFC3339)

					log.Debugf("Runtime updating service: %s", service.Name)
					if err := service.Update(k.client); err != nil {
						log.Debugf("Runtime failed to update service %s: %v", service.Name, err)
						continue
					}
				}
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

	var events <-chan runtime.Event
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
