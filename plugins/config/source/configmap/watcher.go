package configmap

import (
	"errors"
	"time"

	"github.com/asim/go-micro/v3/config/source"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type watcher struct {
	opts      source.Options
	name      string
	namespace string
	client    *kubernetes.Clientset
	st        cache.Store
	ct        cache.Controller
	ch        chan *source.ChangeSet

	exit chan bool
	stop chan struct{}
}

func newWatcher(n, ns string, c *kubernetes.Clientset, opts source.Options) (source.Watcher, error) {
	w := &watcher{
		opts:      opts,
		name:      n,
		namespace: ns,
		client:    c,
		ch:        make(chan *source.ChangeSet),
		exit:      make(chan bool),
		stop:      make(chan struct{}),
	}

	lw := cache.NewListWatchFromClient(w.client.CoreV1().RESTClient(), "configmaps", w.namespace, fields.OneTermEqualSelector("metadata.name", w.name))
	st, ct := cache.NewInformer(
		lw,
		&v12.ConfigMap{},
		time.Second*30,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: w.handle,
		},
	)

	go ct.Run(w.stop)

	w.ct = ct
	w.st = st

	return w, nil
}

func (w *watcher) handle(oldCmp interface{}, newCmp interface{}) {
	if newCmp == nil {
		return
	}

	data := makeMap(newCmp.(*v12.ConfigMap).Data)

	b, err := w.opts.Encoder.Encode(data)
	if err != nil {
		return
	}

	cs := &source.ChangeSet{
		Format:    w.opts.Encoder.String(),
		Source:    w.name,
		Data:      b,
		Timestamp: newCmp.(*v12.ConfigMap).CreationTimestamp.Time,
	}
	cs.Checksum = cs.Sum()

	w.ch <- cs
}

// Next
func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case cs := <-w.ch:
		return cs, nil
	case <-w.exit:
		return nil, errors.New("watcher stopped")
	}
}

// Stop
func (w *watcher) Stop() error {
	select {
	case <-w.exit:
		return nil
	case <-w.stop:
		return nil
	default:
		close(w.exit)
	}
	return nil
}
