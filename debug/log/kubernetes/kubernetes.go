// Package kubernetes is a logger implementing (github.com/micro/go-micro/debug/log).Log
package kubernetes

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/micro/go-micro/debug/log"
	"github.com/micro/go-micro/util/kubernetes/client"
)

type klog struct {
	client client.Kubernetes

	log.Options
}

func (k *klog) Read(options ...log.ReadOption) ([]log.Record, error) {
	opts := &log.ReadOptions{}
	for _, o := range options {
		o(opts)
	}

	r := &client.Resource{
		Kind:  "pod",
		Value: new(client.PodList),
	}
	l := make(map[string]string)
	l["micro"] = "runtime"
	if err := k.client.Get(r, l); err != nil {
		return nil, err
	}

	for _, p := range r.Value.(client.PodList).Items {

	}

	logs, err := k.client.Logs(k.Options.Name)
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	s := bufio.NewScanner(logs)
	records := []log.Record{}
	for s.Scan() {
		line := s.Text()
		record := log.Record{}
		if err := json.Unmarshal(s.Bytes(), &record); err != nil {
			record.Value = line
			record.Metadata = make(map[string]string)
		}
		record.Metadata["service"] = k.Options.Name
		records = append(records, record)
	}
	return records, nil
}

func (k *klog) Write(l log.Record) error {
	return write(l)
}

func (k *klog) Stream() (log.Stream, error) {
	return &klogStreamer{
		streamChan: make(chan log.Record),
		stop:       make(chan bool),
	}, nil
}

// New returns a configured Kubernetes logger
func New(opts ...log.Option) log.Log {
	klog := &klog{}
	for _, o := range opts {
		o(&klog.Options)
	}

	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) > 0 {
		klog.client = client.NewClientInCluster()
	} else {
		klog.client = client.NewLocalDevClient()
	}
	return klog
}
