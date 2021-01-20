package mock

import (
	"encoding/json"
	"sync"

	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/api"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/watch"
)

// Client ...
type Client struct {
	sync.Mutex
	Pods     map[string]*client.Pod
	events   chan watch.Event
	watchers []*mockWatcher
}

// UpdatePod ...
func (m *Client) UpdatePod(podName string, pod *client.Pod) (*client.Pod, error) {
	p, ok := m.Pods[podName]
	if !ok {
		return nil, api.ErrNotFound
	}

	updateMetadata(p.Metadata, pod.Metadata)

	pstr, _ := json.Marshal(p)

	m.events <- watch.Event{
		Type:   watch.Modified,
		Object: json.RawMessage(pstr),
	}

	return nil, nil
}

// ListPods ...
func (m *Client) ListPods(labels map[string]string) (*client.PodList, error) {
	var pods []client.Pod

	for _, v := range m.Pods {
		if labelFilterMatch(v.Metadata.Labels, labels) {
			pods = append(pods, *v)
		}
	}
	return &client.PodList{
		Items: pods,
	}, nil
}

// WatchPods ...
func (m *Client) WatchPods(labels map[string]string) (watch.Watch, error) {
	w := &mockWatcher{
		results: make(chan watch.Event),
		stop:    make(chan bool),
	}

	i := len(m.watchers) // length of watchers is current index
	m.watchers = append(m.watchers, w)

	go func() {
		<-w.stop
		m.watchers = append(m.watchers[:i], m.watchers[i+1:]...)
	}()

	return w, nil
}

// newClient ...
func newClient() client.Kubernetes {
	return &Client{}
}

// NewClient ...
func NewClient() *Client {
	c := &Client{
		Pods:   make(map[string]*client.Pod),
		events: make(chan watch.Event),
	}

	// broadcast events to watchers
	go func() {
		for e := range c.events {
			for _, w := range c.watchers {
				w.results <- e
			}
		}
	}()

	return c
}

// Teardown ...
func Teardown(c *Client) {

	for _, p := range c.Pods {
		pstr, _ := json.Marshal(p)

		c.events <- watch.Event{
			Type:   watch.Deleted,
			Object: json.RawMessage(pstr),
		}
	}

	c.Pods = make(map[string]*client.Pod)
}
