// Package configmap config is an interface for dynamic configuration.
package configmap

import (
	"fmt"

	"github.com/micro/go-micro/v2/config/source"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type configmap struct {
	opts       source.Options
	client     *kubernetes.Clientset
	cerr       error
	name       string
	namespace  string
	configPath string
}

// Predefined variables
var (
	DefaultName       = "micro"
	DefaultConfigPath = ""
	DefaultNamespace  = "default"
)

func (k *configmap) Read() (*source.ChangeSet, error) {
	if k.cerr != nil {
		return nil, k.cerr
	}

	cmp, err := k.client.CoreV1().ConfigMaps(k.namespace).Get(k.name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	data := makeMap(cmp.Data)

	b, err := k.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Format:    k.opts.Encoder.String(),
		Source:    k.String(),
		Data:      b,
		Timestamp: cmp.CreationTimestamp.Time,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

// Write is unsupported
func (k *configmap) Write(cs *source.ChangeSet) error {
	return nil
}

func (k *configmap) String() string {
	return "configmap"
}

func (k *configmap) Watch() (source.Watcher, error) {
	if k.cerr != nil {
		return nil, k.cerr
	}

	w, err := newWatcher(k.name, k.namespace, k.client, k.opts)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// NewSource is a factory function
func NewSource(opts ...source.Option) source.Source {
	var (
		options    = source.NewOptions(opts...)
		name       = DefaultName
		configPath = DefaultConfigPath
		namespace  = DefaultNamespace
	)

	prefix, ok := options.Context.Value(prefixKey{}).(string)
	if ok {
		name = prefix
	}

	cfg, ok := options.Context.Value(configPathKey{}).(string)
	if ok {
		configPath = cfg
	}

	sname, ok := options.Context.Value(nameKey{}).(string)
	if ok {
		name = sname
	}

	ns, ok := options.Context.Value(namespaceKey{}).(string)
	if ok {
		namespace = ns
	}

	// TODO handle if the client fails what to do current return does not support error
	client, err := getClient(configPath)

	return &configmap{
		cerr:       err,
		client:     client,
		opts:       options,
		name:       name,
		configPath: configPath,
		namespace:  namespace,
	}
}
