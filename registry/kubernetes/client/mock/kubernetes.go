// Package mock implements a mock k8s client.
package mock

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"

	"go-micro.dev/v6/registry/kubernetes/client"
	"go-micro.dev/v6/registry/kubernetes/client/api"
	"go-micro.dev/v6/registry/kubernetes/client/watch"
)

// Client ...
type Client struct {
	sync.RWMutex
	Pods     map[string]*client.Pod
	events   chan watch.Event
	watchers []*mockWatcher
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
			c.RLock()
			for _, w := range c.watchers {
				select {
				case <-w.stop:
				default:
					w.results <- e
				}
			}
			c.RUnlock()
		}
	}()

	return c
}

// UpdatePod ...
func (c *Client) UpdatePod(podName string, pod *client.Pod) (*client.Pod, error) {
	if podName == "" {
		return nil, errors.Wrap(api.ErrNoPodName, "failed to update pod")
	}

	p, ok := c.Pods[podName]
	if !ok {
		return nil, api.ErrNotFound
	}

	updateMetadata(p.Metadata, pod.Metadata)

	pstr, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	c.events <- watch.Event{
		Type:   watch.Modified,
		Object: json.RawMessage(pstr),
	}

	//nolint:nilnil
	return nil, nil
}

// ListPods ...
func (c *Client) ListPods(labels map[string]string) (*client.PodList, error) {
	var pods []client.Pod

	for _, v := range c.Pods {
		if labelFilterMatch(v.Metadata.Labels, labels) {
			pods = append(pods, *v)
		}
	}

	p := client.PodList{
		Items: pods,
	}

	return &p, nil
}

// WatchPods ...
func (c *Client) WatchPods(labels map[string]string) (watch.Watch, error) {
	w := &mockWatcher{
		results: make(chan watch.Event),
		stop:    make(chan bool),
	}

	i := len(c.watchers) // length of watchers is current index
	c.Lock()
	c.watchers = append(c.watchers, w)
	c.Unlock()

	go func() {
		<-w.stop

		c.Lock()
		c.watchers = append(c.watchers[:i], c.watchers[i+1:]...)
		c.Unlock()
	}()

	return w, nil
}

// Teardown ...
func Teardown(c *Client) {
	for _, p := range c.Pods {
		//nolint:errcheck
		pstr, _ := json.Marshal(p)

		c.events <- watch.Event{
			Type:   watch.Deleted,
			Object: json.RawMessage(pstr),
		}
	}

	c.Pods = make(map[string]*client.Pod)
}
