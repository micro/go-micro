// Package kubernetes provides a kubernetes registry
package kubernetes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
)

type kregistry struct {
	client  client.Kubernetes
	timeout time.Duration
	options registry.Options
}

var (
	// used on pods as labels & services to select
	// eg: svcSelectorPrefix+"svc.name"
	svcSelectorPrefix = "micro.mu/selector-"
	svcSelectorValue  = "service"

	labelTypeKey          = "micro.mu/type"
	labelTypeValueService = "service"

	// used on k8s services to scope a serialised
	// micro service by pod name
	annotationServiceKeyPrefix = "micro.mu/service-"

	// Pod status
	podRunning = "Running"

	// label name regex
	labelRe = regexp.MustCompilePOSIX("[-A-Za-z0-9_.]")
)

// podSelector
var podSelector = map[string]string{
	labelTypeKey: labelTypeValueService,
}

func init() {
	cmd.DefaultRegistries["kubernetes"] = NewRegistry
}

func configure(k *kregistry, opts ...registry.Option) error {
	for _, o := range opts {
		o(&k.options)
	}

	// get first host
	var host string
	if len(k.options.Addrs) > 0 && len(k.options.Addrs[0]) > 0 {
		host = k.options.Addrs[0]
	}

	if k.options.Timeout == 0 {
		k.options.Timeout = time.Second * 1
	}

	// if no hosts setup, assume InCluster
	var c client.Kubernetes
	if len(host) == 0 {
		c = client.NewClientInCluster()
	} else {
		c = client.NewClientByHost(host)
	}

	k.client = c
	k.timeout = k.options.Timeout

	return nil
}

// serviceName generates a valid service name for k8s labels
func serviceName(name string) string {
	aname := make([]byte, len(name))

	for i, r := range []byte(name) {
		if !labelRe.Match([]byte{r}) {
			aname[i] = '_'
			continue
		}
		aname[i] = r
	}

	return string(aname)
}

// Init allows reconfig of options
func (c *kregistry) Init(opts ...registry.Option) error {
	return configure(c, opts...)
}

// Options returns the registry Options
func (c *kregistry) Options() registry.Options {
	return c.options
}

// Register sets a service selector label and an annotation with a
// serialised version of the service passed in.
func (c *kregistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("you must register at least one node")
	}

	// TODO: grab podname from somewhere better than this.
	podName := os.Getenv("HOSTNAME")
	svcName := s.Name

	// encode micro service
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	svc := string(b)

	pod := &client.Pod{
		Metadata: &client.Meta{
			Labels: map[string]*string{
				labelTypeKey:                             &labelTypeValueService,
				svcSelectorPrefix + serviceName(svcName): &svcSelectorValue,
			},
			Annotations: map[string]*string{
				annotationServiceKeyPrefix + serviceName(svcName): &svc,
			},
		},
	}

	if _, err := c.client.UpdatePod(podName, pod); err != nil {
		return err
	}

	return nil

}

// Deregister nils out any things set in Register
func (c *kregistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("you must deregister at least one node")
	}

	// TODO: grab podname from somewhere better than this.
	podName := os.Getenv("HOSTNAME")
	svcName := s.Name

	pod := &client.Pod{
		Metadata: &client.Meta{
			Labels: map[string]*string{
				svcSelectorPrefix + serviceName(svcName): nil,
			},
			Annotations: map[string]*string{
				annotationServiceKeyPrefix + serviceName(svcName): nil,
			},
		},
	}

	if _, err := c.client.UpdatePod(podName, pod); err != nil {
		return err
	}

	return nil

}

// GetService will get all the pods with the given service selector,
// and build services from the annotations.
func (c *kregistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	pods, err := c.client.ListPods(map[string]string{
		svcSelectorPrefix + serviceName(name): svcSelectorValue,
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, registry.ErrNotFound
	}

	// svcs mapped by version
	svcs := make(map[string]*registry.Service)

	// loop through items
	for _, pod := range pods.Items {
		if pod.Status.Phase != podRunning {
			continue
		}
		// get serialised service from annotation
		svcStr, ok := pod.Metadata.Annotations[annotationServiceKeyPrefix+serviceName(name)]
		if !ok {
			continue
		}

		// unmarshal service string
		var svc registry.Service
		err := json.Unmarshal([]byte(*svcStr), &svc)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal service '%s' from pod annotation", name)
		}

		// merge up pod service & ip with versioned service.
		vs, ok := svcs[svc.Version]
		if !ok {
			svcs[svc.Version] = &svc
			continue
		}

		vs.Nodes = append(vs.Nodes, svc.Nodes...)
	}

	list := make([]*registry.Service, 0, len(svcs))
	for _, val := range svcs {
		list = append(list, val)
	}
	return list, nil
}

// ListServices will list all the service names
func (c *kregistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	pods, err := c.client.ListPods(podSelector)
	if err != nil {
		return nil, err
	}

	// svcs mapped by name
	svcs := make(map[string]bool)

	for _, pod := range pods.Items {
		if pod.Status.Phase != podRunning {
			continue
		}
		for k, v := range pod.Metadata.Annotations {
			if !strings.HasPrefix(k, annotationServiceKeyPrefix) {
				continue
			}

			// we have to unmarshal the annotation itself since the
			// key is encoded to match the regex restriction.
			var svc registry.Service
			if err := json.Unmarshal([]byte(*v), &svc); err != nil {
				continue
			}
			svcs[svc.Name] = true
		}
	}

	var list []*registry.Service
	for val := range svcs {
		list = append(list, &registry.Service{Name: val})
	}
	return list, nil
}

// Watch returns a kubernetes watcher
func (c *kregistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newWatcher(c, opts...)
}

func (c *kregistry) String() string {
	return "kubernetes"
}

// NewRegistry creates a kubernetes registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	k := &kregistry{
		options: registry.Options{},
	}
	configure(k, opts...)
	return k
}
