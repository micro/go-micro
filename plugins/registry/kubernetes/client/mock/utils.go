package mock

import (
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/watch"
)

type mockWatcher struct {
	results chan watch.Event
	stop    chan bool
}

// Changes returns the results channel
func (w *mockWatcher) ResultChan() <-chan watch.Event {
	return w.results
}

// Stop closes any channels
func (w *mockWatcher) Stop() {
	select {
	case <-w.stop:
		return
	default:
		close(w.stop)
		close(w.results)
	}
}

func updateMetadata(a, b *client.Meta) {
	if a == nil || b == nil {
		return
	}

	if b.Labels != nil {
		for lk, lv := range b.Labels {
			labels := a.Labels
			if lv == nil {
				delete(labels, lk)
				continue
			}
			labels[lk] = lv
		}
	}

	if b.Annotations != nil {
		for ak, av := range b.Annotations {
			ann := a.Annotations
			if av == nil {
				delete(ann, ak)
				continue
			}
			ann[ak] = av
		}
	}
}

func labelFilterMatch(a map[string]*string, b map[string]string) bool {
	match := true
	for lk, lv := range b {
		ml, ok := a[lk]
		if !ok || *ml != lv {
			match = false
			break
		}
	}
	return match
}
