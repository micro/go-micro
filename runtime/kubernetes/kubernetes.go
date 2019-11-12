// Package kubernetes implements kubernetes micro runtime
package kubernetes

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/micro/go-micro/runtime"
	"github.com/micro/go-micro/runtime/kubernetes/client"
	"github.com/micro/go-micro/util/log"
)

type kubernetes struct {
	sync.RWMutex
	// options configure runtime
	options runtime.Options
	// indicates if we're running
	running bool
	// used to start new services
	start chan *runtime.Service
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
		start:   make(chan *runtime.Service, 128),
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

// Registers a service
func (k *kubernetes) Create(s *runtime.Service, opts ...runtime.CreateOption) error {
	k.Lock()
	defer k.Unlock()

	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}

	// push into start queue
	k.start <- s

	return nil
}

// Get returns all instances of given service
func (k *kubernetes) Get(opts ...runtime.GetOption) ([]*runtime.Service, error) {
	// TODO: implement this
	return nil, nil
}

// Update the service in place
func (k *kubernetes) Update(s *runtime.Service) error {
	// parse version into human readable timestamp
	updateTimeStamp, err := strconv.ParseInt(s.Version, 10, 64)
	if err != nil {
		return err
	}
	unixTimeUTC := time.Unix(updateTimeStamp, 0)

	d := &client.Deployment{
		Spec: &client.DeploymentSpec{
			Template: &client.Template{
				Metadata: &client.Metadata{
					Annotations: map[string]string{
						"build": unixTimeUTC.Format(time.RFC3339),
					},
				},
			},
		},
	}

	return k.client.UpdateDeployment(d)
}

// Remove a service
func (k *kubernetes) Delete(s *runtime.Service) error {
	k.Lock()
	defer k.Unlock()

	// TODO:
	// * delete service
	// * delete dpeloyment

	return nil
}

// List the managed services
func (k *kubernetes) List() ([]*runtime.Service, error) {
	// list all micro core deployments
	deployments, err := k.client.ListDeployments()
	if err != nil {
		return nil, err
	}

	log.Debugf("Runtime found %d micro deployment", len(deployments.Items))

	services := make([]*runtime.Service, 0, len(deployments.Items))

	for _, service := range deployments.Items {
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
			// check running services
			services, err := k.List()
			if err != nil {
				log.Debugf("Runtime failed listing running services: %v", err)
				continue
			}
			// TODO: for now we just log the running services
			// * make sure all core deployments exist
			// * make sure all core services are exposed
			for _, service := range services {
				log.Debugf("Runtime found running service: %v", service)
			}
		case service := <-k.start:
			// TODO: this is a noop for now
			// * create a deployment
			// * expose a service
			log.Debugf("Runtime starting service: %s", service.Name)
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
				buildTime := time.Unix(updateTimeStamp, 0)
				log.Debugf("build time: %s", buildTime)
				if len(event.Service) > 0 {
					// TODO:
					continue
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
