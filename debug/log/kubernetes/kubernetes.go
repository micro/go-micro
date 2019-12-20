// Package kubernetes is a logger implementing (github.com/micro/go-micro/debug/log).Log
package kubernetes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

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

	logsToGet, err := k.getMatchingPods()
	if err != nil {
		return nil, err
	}
	records := []log.Record{}

	for _, l := range logsToGet {
		logParams := make(map[string]string)
		if !opts.Since.Equal(time.Time{}) {
			logParams["sinceSeconds"] = strconv.Itoa(int(time.Since(opts.Since).Seconds()))
		}
		if opts.Count != 0 {
			logParams["tailLines"] = strconv.Itoa(opts.Count)
		}
		if opts.Stream == true {
			logParams["follow"] = "true"
		}
		logs, err := k.client.Logs(l, client.AdditionalParams(logParams))
		if err != nil {
			return nil, err
		}
		defer logs.Close()
		s := bufio.NewScanner(logs)
		for s.Scan() {
			record := k.parse(s.Text())
			record.Metadata["pod"] = l
			records = append(records, record)
		}
	}

	sort.Sort(byTimestamp(records))
	return records, nil
}

func (k *klog) Write(l log.Record) error {
	return write(l)
}

func (k *klog) Stream() (log.Stream, error) {
	return k.stream()
}

func (k *klog) stream() (log.Stream, error) {
	pods, err := k.getMatchingPods()
	if err != nil {
		return nil, err
	}
	logStreamer := &klogStreamer{
		streamChan: make(chan log.Record),
		stop:       make(chan bool),
	}
	errorChan := make(chan error)
	go func(stopChan <-chan bool) {
		for {
			select {
			case <-stopChan:
				return
			case err := <-errorChan:
				fmt.Fprintf(os.Stderr, err.Error())
			}
		}
	}(logStreamer.stop)
	for _, pod := range pods {
		go k.individualPodLogStreamer(pod, logStreamer.streamChan, errorChan, logStreamer.stop)
	}
	return logStreamer, nil
}

func (k *klog) individualPodLogStreamer(podName string, recordChan chan<- log.Record, errorChan chan<- error, stopChan <-chan bool) {
	p := make(map[string]string)
	p["follow"] = "true"
	body, err := k.client.Logs(podName, client.AdditionalParams(p))
	if err != nil {
		errorChan <- err
		return
	}
	s := bufio.NewScanner(body)
	defer body.Close()
	for {
		select {
		case <-stopChan:
			return
		default:
			if s.Scan() {
				record := k.parse(s.Text())
				recordChan <- record
			} else {
				time.Sleep(time.Second)
			}
		}
	}
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

func (k *klog) getMatchingPods() ([]string, error) {
	r := &client.Resource{
		Kind:  "pod",
		Value: new(client.PodList),
	}
	l := make(map[string]string)
	l["micro"] = "runtime"
	if err := k.client.Get(r, l); err != nil {
		return nil, err
	}

	var matches []string
	for _, p := range r.Value.(*client.PodList).Items {
		if p.Metadata.Labels["name"] == k.Options.Name {
			matches = append(matches, p.Metadata.Name)
		}
	}
	return matches, nil
}

func (k *klog) parse(line string) log.Record {
	record := log.Record{}
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		record.Timestamp = time.Now().UTC()
		record.Message = line
		record.Metadata = make(map[string]string)
	}
	record.Metadata["service"] = k.Options.Name
	return record
}
