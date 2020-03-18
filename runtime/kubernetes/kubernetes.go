// Package kubernetes implements kubernetes micro runtime
package kubernetes

import (
	"sync"
	"time"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/runtime"
	"github.com/micro/go-micro/v2/util/kubernetes/client"
)

// action to take on runtime service
type action int

type kubernetes struct {
	sync.RWMutex
	// options configure runtime
	options runtime.Options
	// indicates if we're running
	running bool
	// used to stop the runtime
	closed chan bool
	// client is kubernetes client
	client client.Client
}

// getService queries kubernetes for micro service
// NOTE: this function is not thread-safe
func (k *kubernetes) getService(labels map[string]string) ([]*service, error) {
	// get the service status
	serviceList := new(client.ServiceList)
	r := &client.Resource{
		Kind:  "service",
		Value: serviceList,
	}

	// get the service from k8s
	if err := k.client.Get(r, labels); err != nil {
		return nil, err
	}

	// get the deployment status
	depList := new(client.DeploymentList)
	d := &client.Resource{
		Kind:  "deployment",
		Value: depList,
	}
	if err := k.client.Get(d, labels); err != nil {
		return nil, err
	}

	// get the pods from k8s
	podList := new(client.PodList)
	p := &client.Resource{
		Kind:  "pod",
		Value: podList,
	}
	if err := k.client.Get(p, labels); err != nil {
		return nil, err
	}

	// service map
	svcMap := make(map[string]*service)

	// collect info from kubernetes service
	for _, kservice := range serviceList.Items {
		// name of the service
		name := kservice.Metadata.Labels["name"]
		// version of the service
		version := kservice.Metadata.Labels["version"]

		// save as service
		svcMap[name+version] = &service{
			Service: &runtime.Service{
				Name:     name,
				Version:  version,
				Metadata: make(map[string]string),
			},
			kservice: &kservice,
		}

		// copy annotations metadata into service metadata
		for k, v := range kservice.Metadata.Annotations {
			svcMap[name+version].Service.Metadata[k] = v
		}
	}

	// collect additional info from kubernetes deployment
	for _, kdep := range depList.Items {
		// name of the service
		name := kdep.Metadata.Labels["name"]
		// versio of the service
		version := kdep.Metadata.Labels["version"]

		// access existing service map based on name + version
		if svc, ok := svcMap[name+version]; ok {
			// we're expecting our own service name in metadata
			if _, ok := kdep.Metadata.Annotations["name"]; !ok {
				continue
			}

			// set the service name, version and source
			// based on existing annotations we stored
			svc.Service.Name = kdep.Metadata.Annotations["name"]
			svc.Service.Version = kdep.Metadata.Annotations["version"]
			svc.Service.Source = kdep.Metadata.Annotations["source"]

			// delete from metadata
			delete(kdep.Metadata.Annotations, "name")
			delete(kdep.Metadata.Annotations, "version")
			delete(kdep.Metadata.Annotations, "source")

			// copy all annotations metadata into service metadata
			for k, v := range kdep.Metadata.Annotations {
				svc.Service.Metadata[k] = v
			}

			// get the status from the pods
			var status string

			for _, item := range podList.Items {
				switch item.Status.Phase {
				case "Failed":
					status = item.Status.Reason
				default:
					status = item.Status.Phase
				}
			}

			// unknown status
			if len(status) == 0 {
				status = "n/a"
			}

			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime setting %s service deployment status: %v", name, status)
			}
			svc.Service.Metadata["status"] = status
			// save deployment
			svc.kdeploy = &kdep
		}
	}

	// collect all the services and return
	services := make([]*service, 0, len(serviceList.Items))

	for _, service := range svcMap {
		services = append(services, service)
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
		case event := <-events:
			// NOTE: we only handle Update events for now
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime received notification event: %v", event)
			}
			switch event.Type {
			case runtime.Update:
				// only process if there's an actual service
				// we do not update all the things individually
				if len(event.Service) == 0 {
					continue
				}

				// format the name
				name := client.Format(event.Service)

				// set the default labels
				labels := map[string]string{
					"micro": k.options.Type,
					"name":  name,
				}

				if len(event.Version) > 0 {
					labels["version"] = event.Version
				}

				// get the deployment status
				deployed := new(client.DeploymentList)

				// get the existing service rather than creating a new one
				err := k.client.Get(&client.Resource{
					Kind:  "deployment",
					Value: deployed,
				}, labels)

				if err != nil {
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("Runtime update failed to get service %s: %v", event.Service, err)
					}
					continue
				}

				// technically we should not receive multiple versions but hey ho
				for _, service := range deployed.Items {
					// check the name matches
					if service.Metadata.Name != name {
						continue
					}

					// update build time annotation
					if service.Spec.Template.Metadata.Annotations == nil {
						service.Spec.Template.Metadata.Annotations = make(map[string]string)

					}

					// check the existing build timestamp
					if build, ok := service.Spec.Template.Metadata.Annotations["build"]; ok {
						buildTime, err := time.Parse(time.RFC3339, build)
						if err == nil && !event.Timestamp.After(buildTime) {
							continue
						}
					}

					// update the build time
					service.Spec.Template.Metadata.Annotations["build"] = event.Timestamp.Format(time.RFC3339)
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("Runtime updating service: %s deployment: %s", event.Service, service.Metadata.Name)
					}
					if err := k.client.Update(deploymentResource(&service)); err != nil {
						if logger.V(logger.DebugLevel, logger.DefaultLogger) {
							logger.Debugf("Runtime failed to update service %s: %v", event.Service, err)
						}
						continue
					}
				}
			}
		case <-k.closed:
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime stopped")
			}
			return
		}
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

	options := runtime.CreateOptions{
		Type: k.options.Type,
	}
	for _, o := range opts {
		o(&options)
	}

	// default type if it doesn't exist
	if len(options.Type) == 0 {
		options.Type = k.options.Type
	}

	// default the source if it doesn't exist
	if len(s.Source) == 0 {
		s.Source = k.options.Source
	}

	// determine the image from the source and options
	options.Image = k.getImage(s, options)

	// create new service
	service := newService(s, options)

	// start the service
	return service.Start(k.client)
}

// Read returns all instances of given service
func (k *kubernetes) Read(opts ...runtime.ReadOption) ([]*runtime.Service, error) {
	k.Lock()
	defer k.Unlock()

	// set the default labels
	labels := map[string]string{
		"micro": k.options.Type,
	}

	var options runtime.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	if len(options.Service) > 0 {
		labels["name"] = client.Format(options.Service)
	}

	// add version to labels if a version has been supplied
	if len(options.Version) > 0 {
		labels["version"] = options.Version
	}

	if len(options.Type) > 0 {
		labels["micro"] = options.Type
	}

	srvs, err := k.getService(labels)
	if err != nil {
		return nil, err
	}

	var services []*runtime.Service
	for _, service := range srvs {
		services = append(services, service.Service)
	}

	return services, nil
}

// List the managed services
func (k *kubernetes) List() ([]*runtime.Service, error) {
	k.Lock()
	defer k.Unlock()

	labels := map[string]string{
		"micro": k.options.Type,
	}

	if logger.V(logger.DebugLevel, logger.DefaultLogger) {
		logger.Debugf("Runtime listing all micro services")
	}

	srvs, err := k.getService(labels)
	if err != nil {
		return nil, err
	}

	var services []*runtime.Service
	for _, service := range srvs {
		services = append(services, service.Service)
	}

	return services, nil
}

// Update the service in place
func (k *kubernetes) Update(s *runtime.Service) error {
	// get the existing service
	// set the default labels
	labels := map[string]string{
		"micro": k.options.Type,
	}

	if len(s.Name) > 0 {
		labels["name"] = client.Format(s.Name)
	}

	if len(s.Version) > 0 {
		labels["version"] = s.Version
	}

	// get the existing service
	services, err := k.getService(labels)
	if err != nil {
		return err
	}

	// update the relevant services
	for _, service := range services {
		// nil check
		if service.kdeploy.Metadata == nil || service.kdeploy.Metadata.Annotations == nil {
			md := new(client.Metadata)
			md.Annotations = make(map[string]string)
			service.kdeploy.Metadata = md
		}

		// update metadata
		for k, v := range s.Metadata {
			service.kdeploy.Metadata.Annotations[k] = v
		}
		// update build time annotation
		service.kdeploy.Spec.Template.Metadata.Annotations["build"] = time.Now().Format(time.RFC3339)

		// update the service
		if err := service.Update(k.client); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes a service
func (k *kubernetes) Delete(s *runtime.Service) error {
	k.Lock()
	defer k.Unlock()

	// create new kubernetes micro service
	service := newService(s, runtime.CreateOptions{
		Type: k.options.Type,
	})

	return service.Stop(k.client)
}

// Start starts the runtime
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
	if k.options.Scheduler != nil {
		var err error
		events, err = k.options.Scheduler.Notify()
		if err != nil {
			// TODO: should we bail here?
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("Runtime failed to start update notifier")
			}
		}
	}

	go k.run(events)

	return nil
}

// Stop shuts down the runtime
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
		// stop the scheduler
		if k.options.Scheduler != nil {
			return k.options.Scheduler.Close()
		}
	}

	return nil
}

// String implements stringer interface
func (k *kubernetes) String() string {
	return "kubernetes"
}

// NewRuntime creates new kubernetes runtime
func NewRuntime(opts ...runtime.Option) runtime.Runtime {
	// get default options
	options := runtime.Options{
		// Create labels with type "micro": "service"
		Type: "service",
	}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// kubernetes client
	client := client.NewClusterClient()

	return &kubernetes{
		options: options,
		closed:  make(chan bool),
		client:  client,
	}
}

func (k *kubernetes) getImage(s *runtime.Service, options runtime.CreateOptions) string {
	// use the image when its specified
	if len(options.Image) > 0 {
		return options.Image
	}

	if len(k.options.Image) > 0 {
		return k.options.Image
	}

	return ""
}
