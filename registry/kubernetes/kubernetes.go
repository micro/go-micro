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

	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/util/kubernetes/client"
)

type kregistry struct {
	client  client.Client
	timeout time.Duration
	options registry.Options
}

var (
	// used on pods as labels & services to select
	// eg: svcSelectorPrefix+"svc.name"
	servicePrefix = "go.micro/"
	serviceValue  = "service"

	labelTypeKey   = "micro"
	labelTypeValue = "service"

	// used on k8s services to scope a serialised
	// micro service by pod name
	annotationPrefix = "go.micro/"

	// Pod status
	podRunning = "Running"

	// label name regex
	labelRe = regexp.MustCompilePOSIX("[-A-Za-z0-9_.]")
)

// podSelector
var podSelector = map[string]string{
	labelTypeKey: labelTypeValue,
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
	var c client.Client

	if len(host) > 0 {
		c = client.NewLocalClient(host)
	} else {
		c = client.NewClusterClient()
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
		return errors.New("no nodes")
	}

	// TODO: grab podname from somewhere better than this.
	podName := os.Getenv("HOSTNAME")
	svcName := s.Name

	// encode micro service
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	/// marshalled service
	svc := string(b)

	pod := &client.Pod{
		Metadata: &client.Metadata{
			Labels: map[string]string{
				// micro: service
				labelTypeKey: labelTypeValue,
				// micro/service/name: service
				servicePrefix + serviceName(svcName): serviceValue,
			},
			Annotations: map[string]string{
				// micro/service/name: definition
				annotationPrefix + serviceName(svcName): svc,
			},
		},
	}

	return c.client.Update(&client.Resource{
		Name:  podName,
		Kind:  "pod",
		Value: pod,
	})
}

// Deregister nils out any things set in Register
func (c *kregistry) Deregister(s *registry.Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("you must deregister at least one node")
	}

	// TODO: grab podname from somewhere better than this.
	podName := os.Getenv("HOSTNAME")
	svcName := s.Name

	pod := &client.Pod{
		Metadata: &client.Metadata{
			Labels: map[string]string{
				servicePrefix + serviceName(svcName): "",
			},
			Annotations: map[string]string{
				annotationPrefix + serviceName(svcName): "",
			},
		},
	}

	return c.client.Update(&client.Resource{
		Name:  podName,
		Kind:  "pod",
		Value: pod,
	})
}

// GetService will get all the pods with the given service selector,
// and build services from the annotations.
func (c *kregistry) GetService(name string) ([]*registry.Service, error) {
	var pods client.PodList

	if err := c.client.Get(&client.Resource{
		Kind:  "pod",
		Value: &pods,
	}, map[string]string{
		servicePrefix + serviceName(name): serviceValue,
	}); err != nil {
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
		svcStr, ok := pod.Metadata.Annotations[annotationPrefix+serviceName(name)]
		if !ok {
			continue
		}

		// unmarshal service string
		var svc registry.Service

		if err := json.Unmarshal([]byte(svcStr), &svc); err != nil {
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
func (c *kregistry) ListServices() ([]*registry.Service, error) {
	var pods client.PodList

	if err := c.client.Get(&client.Resource{
		Kind:  "pod",
		Value: &pods,
	}, podSelector); err != nil {
		return nil, err
	}

	// svcs mapped by name
	svcs := make(map[string]bool)

	for _, pod := range pods.Items {
		if pod.Status.Phase != podRunning {
			continue
		}
		for k, v := range pod.Metadata.Annotations {
			if !strings.HasPrefix(k, annotationPrefix) {
				continue
			}

			// we have to unmarshal the annotation itself since the
			// key is encoded to match the regex restriction.
			var svc registry.Service

			if err := json.Unmarshal([]byte(v), &svc); err != nil {
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
