// Package kubernetes implements kubernetes micro runtime
package kubernetes

import (
	"fmt"
	"sync"
	"time"

	log "github.com/micro/go-micro/v2/logger"
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
	// namespaces which exist
	namespaces []client.Namespace
}

// namespaceExists returns a boolean indicating if a namespace exists
func (k *kubernetes) namespaceExists(name string) (bool, error) {
	// populate the cache
	if k.namespaces == nil {
		namespaceList := new(client.NamespaceList)
		resource := &client.Resource{Kind: "namespace", Value: namespaceList}
		if err := k.client.List(resource); err != nil {
			return false, err
		}
		k.namespaces = namespaceList.Items
	}

	// check if the namespace exists in the cache
	for _, n := range k.namespaces {
		if n.Metadata.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// createNamespace creates a new k8s namespace
func (k *kubernetes) createNamespace(namespace string) error {
	ns := client.Namespace{Metadata: &client.Metadata{Name: namespace}}
	err := k.client.Create(&client.Resource{Kind: "namespace", Value: ns})

	// add to cache
	if err == nil && k.namespaces != nil {
		k.namespaces = append(k.namespaces, ns)
	}

	return err
}

// getService queries kubernetes for micro service
// NOTE: this function is not thread-safe
func (k *kubernetes) getService(labels map[string]string, opts ...client.GetOption) ([]*service, error) {
	// get the service status
	serviceList := new(client.ServiceList)
	r := &client.Resource{
		Kind:  "service",
		Value: serviceList,
	}

	opts = append(opts, client.GetLabels(labels))

	// get the service from k8s
	if err := k.client.Get(r, opts...); err != nil {
		return nil, err
	}

	// get the deployment status
	depList := new(client.DeploymentList)
	d := &client.Resource{
		Kind:  "deployment",
		Value: depList,
	}
	if err := k.client.Get(d, opts...); err != nil {
		return nil, err
	}

	// get the pods from k8s
	podList := new(client.PodList)
	p := &client.Resource{
		Kind:  "pod",
		Value: podList,
	}
	if err := k.client.Get(p, opts...); err != nil {
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

		srv := &service{
			Service: &runtime.Service{
				Name:     name,
				Version:  version,
				Metadata: make(map[string]string),
			},
			kservice: &kservice,
		}

		// set the address
		address := kservice.Spec.ClusterIP
		port := kservice.Spec.Ports[0]
		srv.Service.Metadata["address"] = fmt.Sprintf("%s:%d", address, port.Port)
		// set the type of service
		srv.Service.Metadata["type"] = kservice.Metadata.Labels["micro"]

		// copy annotations metadata into service metadata
		for k, v := range kservice.Metadata.Annotations {
			srv.Service.Metadata[k] = v
		}

		// save as service
		svcMap[name+version] = srv
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

			// parse out deployment status and inject into service metadata
			if len(kdep.Status.Conditions) > 0 {
				svc.Metadata["status"] = kdep.Status.Conditions[0].Type
				svc.Metadata["started"] = kdep.Status.Conditions[0].LastUpdateTime
				delete(svc.Metadata, "error")
			} else {
				svc.Metadata["status"] = "n/a"
			}

			// get the real status
			for _, item := range podList.Items {
				var status string

				// check the name
				if item.Metadata.Labels["name"] != name {
					continue
				}
				// check the version
				if item.Metadata.Labels["version"] != version {
					continue
				}

				switch item.Status.Phase {
				case "Failed":
					status = item.Status.Reason
				default:
					status = item.Status.Phase
				}

				// skip if we can't get the container
				if len(item.Status.Containers) == 0 {
					continue
				}

				// now try get a deeper status
				state := item.Status.Containers[0].State

				// set start time
				if state.Running != nil {
					svc.Metadata["started"] = state.Running.Started
				}

				// set status from waiting
				if v := state.Waiting; v != nil {
					if len(v.Reason) > 0 {
						status = v.Reason
					}
				}
				// TODO: set from terminated

				svc.Metadata["status"] = status
			}

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
			if log.V(log.DebugLevel, log.DefaultLogger) {
				log.Debugf("Runtime received notification event: %v", event)
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
				}, client.GetLabels(labels))

				if err != nil {
					if log.V(log.DebugLevel, log.DefaultLogger) {
						log.Debugf("Runtime update failed to get service %s: %v", event.Service, err)
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

					// update the build time
					service.Spec.Template.Metadata.Annotations["updated"] = fmt.Sprintf("%d", event.Timestamp.Unix())

					if log.V(log.DebugLevel, log.DefaultLogger) {
						log.Debugf("Runtime updating service: %s deployment: %s", event.Service, service.Metadata.Name)
					}
					if err := k.client.Update(deploymentResource(&service)); err != nil {
						if log.V(log.DebugLevel, log.DefaultLogger) {
							log.Debugf("Runtime failed to update service %s: %v", event.Service, err)
						}
						continue
					}
				}
			}
		case <-k.closed:
			if log.V(log.DebugLevel, log.DefaultLogger) {
				log.Debugf("Runtime stopped")
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

func (k *kubernetes) Logs(s *runtime.Service, options ...runtime.LogsOption) (runtime.LogStream, error) {
	klo := newLog(k.client, s.Name, options...)
	stream, err := klo.Stream()
	if err != nil {
		return nil, err
	}
	// If requested, also read existing records and stream those too
	if klo.options.Count > 0 {
		go func() {
			records, err := klo.Read()
			if err != nil {
				log.Errorf("Failed to get logs for service '%v' from k8s: %v", err)
				return
			}
			// @todo: this might actually not run before podLogStream starts
			// and might cause out of order log retrieval at the receiving end.
			// A better approach would probably to suppor this inside the `klog.Stream` method.
			for _, record := range records {
				stream.Chan() <- record
			}
		}()
	}
	return stream, nil
}

type kubeStream struct {
	// the k8s log stream
	stream chan runtime.LogRecord
	// the stop chan
	sync.Mutex
	stop chan bool
	err  error
}

func (k *kubeStream) Error() error {
	return k.err
}

func (k *kubeStream) Chan() chan runtime.LogRecord {
	return k.stream
}

func (k *kubeStream) Stop() error {
	k.Lock()
	defer k.Unlock()
	select {
	case <-k.stop:
		return nil
	default:
		close(k.stop)
		close(k.stream)
	}
	return nil
}

// Creates a service
func (k *kubernetes) Create(s *runtime.Service, opts ...runtime.CreateOption) error {
	k.Lock()
	defer k.Unlock()

	options := runtime.CreateOptions{
		Type:      k.options.Type,
		Namespace: client.DefaultNamespace,
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

	// ensure the namespace exists
	namespace := client.SerializeResourceName(options.Namespace)
	// only do this if the namespace is not default
	if namespace != "default" {
		if exist, err := k.namespaceExists(namespace); err == nil && !exist {
			if err := k.createNamespace(namespace); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// determine the image from the source and options
	options.Image = k.getImage(s, options)

	// create new service
	service := newService(s, options)

	// start the service
	return service.Start(k.client, client.CreateNamespace(options.Namespace))
}

// Read returns all instances of given service
func (k *kubernetes) Read(opts ...runtime.ReadOption) ([]*runtime.Service, error) {
	k.Lock()
	defer k.Unlock()

	// set the default labels
	labels := map[string]string{}

	options := runtime.ReadOptions{
		Namespace: client.DefaultNamespace,
	}

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

	srvs, err := k.getService(labels, client.GetNamespace(options.Namespace))
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
func (k *kubernetes) Update(s *runtime.Service, opts ...runtime.UpdateOption) error {
	options := runtime.UpdateOptions{
		Namespace: client.DefaultNamespace,
	}

	for _, o := range opts {
		o(&options)
	}

	labels := map[string]string{}

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
		service.kdeploy.Spec.Template.Metadata.Annotations["updated"] = fmt.Sprintf("%d", time.Now().Unix())

		// update the service
		if err := service.Update(k.client, client.UpdateNamespace(options.Namespace)); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes a service
func (k *kubernetes) Delete(s *runtime.Service, opts ...runtime.DeleteOption) error {
	options := runtime.DeleteOptions{
		Namespace: client.DefaultNamespace,
	}

	for _, o := range opts {
		o(&options)
	}

	k.Lock()
	defer k.Unlock()

	// create new kubernetes micro service
	service := newService(s, runtime.CreateOptions{
		Type:      k.options.Type,
		Namespace: options.Namespace,
	})

	return service.Stop(k.client, client.DeleteNamespace(options.Namespace))
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
			if log.V(log.DebugLevel, log.DefaultLogger) {
				log.Debugf("Runtime failed to start update notifier")
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
