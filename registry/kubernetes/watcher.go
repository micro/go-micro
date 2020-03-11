package kubernetes

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/util/kubernetes/client"
)

type k8sWatcher struct {
	registry *kregistry
	watcher  client.Watcher
	next     chan *registry.Result
	stop     chan bool

	sync.RWMutex
	pods map[string]*client.Pod
}

// build a cache of pods when the watcher starts.
func (k *k8sWatcher) updateCache() ([]*registry.Result, error) {
	var pods client.PodList

	if err := k.registry.client.Get(&client.Resource{
		Kind:  "pod",
		Value: &pods,
	}, podSelector); err != nil {
		return nil, err
	}

	var results []*registry.Result

	for _, pod := range pods.Items {
		rslts := k.buildPodResults(&pod, nil)

		for _, r := range rslts {
			results = append(results, r)
		}

		k.Lock()
		k.pods[pod.Metadata.Name] = &pod
		k.Unlock()
	}

	return results, nil
}

// look through pod annotations, compare against cache if present
// and return a list of results to send down the wire.
func (k *k8sWatcher) buildPodResults(pod *client.Pod, cache *client.Pod) []*registry.Result {
	var results []*registry.Result
	ignore := make(map[string]bool)

	if pod.Metadata != nil {
		for ak, av := range pod.Metadata.Annotations {
			// check this annotation kv is a service notation
			if !strings.HasPrefix(ak, annotationPrefix) {
				continue
			}

			if len(av) == 0 {
				continue
			}

			// ignore when we check the cached annotations
			// as we take care of it here
			ignore[ak] = true

			// compare aginst cache.
			var cacheExists bool
			var cav string

			if cache != nil && cache.Metadata != nil {
				cav, cacheExists = cache.Metadata.Annotations[ak]
				if cacheExists && len(cav) > 0 && cav == av {
					// service notation exists and is identical -
					// no change result required.
					continue
				}
			}

			rslt := &registry.Result{}
			if cacheExists {
				rslt.Action = "update"
			} else {
				rslt.Action = "create"
			}

			// unmarshal service notation from annotation value
			err := json.Unmarshal([]byte(av), &rslt.Service)
			if err != nil {
				continue
			}

			results = append(results, rslt)
		}
	}

	// loop through cache annotations to find services
	// not accounted for above, and "delete" them.
	if cache != nil && cache.Metadata != nil {
		for ak, av := range cache.Metadata.Annotations {
			if ignore[ak] {
				continue
			}

			// check this annotation kv is a service notation
			if !strings.HasPrefix(ak, annotationPrefix) {
				continue
			}

			rslt := &registry.Result{Action: "delete"}
			// unmarshal service notation from annotation value
			err := json.Unmarshal([]byte(av), &rslt.Service)
			if err != nil {
				continue
			}

			results = append(results, rslt)
		}
	}

	return results
}

// handleEvent will taken an event from the k8s pods API and do the correct
// things with the result, based on the local cache.
func (k *k8sWatcher) handleEvent(event client.Event) {
	var pod client.Pod
	if err := json.Unmarshal([]byte(event.Object), &pod); err != nil {
		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Info("K8s Watcher: Couldnt unmarshal event object from pod")
		}
		return
	}

	switch event.Type {
	case client.Modified:
		// Pod was modified

		k.RLock()
		cache := k.pods[pod.Metadata.Name]
		k.RUnlock()

		// service could have been added, edited or removed.
		var results []*registry.Result

		if pod.Status.Phase == podRunning {
			results = k.buildPodResults(&pod, cache)
		} else {
			// passing in cache might not return all results
			results = k.buildPodResults(&pod, nil)
		}

		for _, result := range results {
			// pod isnt running
			if pod.Status.Phase != podRunning {
				result.Action = "delete"
			}

			select {
			case k.next <- result:
			case <-k.stop:
				return
			}
		}

		k.Lock()
		k.pods[pod.Metadata.Name] = &pod
		k.Unlock()
		return

	case client.Deleted:
		// Pod was deleted
		// passing in cache might not return all results
		results := k.buildPodResults(&pod, nil)

		for _, result := range results {
			result.Action = "delete"
			select {
			case k.next <- result:
			case <-k.stop:
				return
			}
		}

		k.Lock()
		delete(k.pods, pod.Metadata.Name)
		k.Unlock()
		return
	}

}

// Next will block until a new result comes in
func (k *k8sWatcher) Next() (*registry.Result, error) {
	select {
	case r := <-k.next:
		return r, nil
	case <-k.stop:
		return nil, errors.New("watcher stopped")
	}
}

// Stop will cancel any requests, and close channels
func (k *k8sWatcher) Stop() {
	select {
	case <-k.stop:
		return
	default:
		k.watcher.Stop()
		close(k.stop)
	}
}

func newWatcher(kr *kregistry, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	selector := podSelector
	if len(wo.Service) > 0 {
		selector = map[string]string{
			servicePrefix + serviceName(wo.Service): serviceValue,
		}
	}

	// Create watch request
	watcher, err := kr.client.Watch(&client.Resource{
		Kind: "pod",
	}, client.WatchParams(selector))
	if err != nil {
		return nil, err
	}

	k := &k8sWatcher{
		registry: kr,
		watcher:  watcher,
		next:     make(chan *registry.Result),
		stop:     make(chan bool),
		pods:     make(map[string]*client.Pod),
	}

	// update cache, but dont emit changes
	if _, err := k.updateCache(); err != nil {
		return nil, err
	}

	// range over watch request changes, and invoke
	// the update event
	go func() {
		for event := range watcher.Chan() {
			k.handleEvent(event)
		}
	}()

	return k, nil
}
