// Package kubernetes is a logger implementing (github.com/micro/go-micro/v2/debug/log).Log
package kubernetes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/micro/go-micro/v2/debug/log"
	"github.com/micro/go-micro/v2/util/kubernetes/client"
)

type klog struct {
	client client.Client

	log.Options
}

func (k *klog) podLogStream(podName string, stream *kubeStream) {
	p := make(map[string]string)
	p["follow"] = "true"

	// get the logs for the pod
	body, err := k.client.Log(&client.Resource{
		Name: podName,
		Kind: "pod",
	}, client.LogParams(p))

	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	}

	s := bufio.NewScanner(body)
	defer body.Close()

	for {
		select {
		case <-stream.stop:
			return
		default:
			if s.Scan() {
				record := k.parse(s.Text())
				stream.stream <- record
			} else {
				// TODO: is there a blocking call
				// rather than a sleep loop?
				time.Sleep(time.Second)
			}
		}
	}
}

func (k *klog) getMatchingPods() ([]string, error) {
	r := &client.Resource{
		Kind:  "pod",
		Value: new(client.PodList),
	}

	l := make(map[string]string)

	l["name"] = client.Format(k.Options.Name)
	// TODO: specify micro:service
	// l["micro"] = "service"

	if err := k.client.Get(r, client.GetLabels(l)); err != nil {
		return nil, err
	}

	var matches []string

	for _, p := range r.Value.(*client.PodList).Items {
		// find labels that match the name
		if p.Metadata.Labels["name"] == client.Format(k.Options.Name) {
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

func (k *klog) Read(options ...log.ReadOption) ([]log.Record, error) {
	opts := &log.ReadOptions{}
	for _, o := range options {
		o(opts)
	}

	pods, err := k.getMatchingPods()
	if err != nil {
		return nil, err
	}

	var records []log.Record

	for _, pod := range pods {
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

		logs, err := k.client.Log(&client.Resource{
			Name: pod,
			Kind: "pod",
		}, client.LogParams(logParams))

		if err != nil {
			return nil, err
		}
		defer logs.Close()

		s := bufio.NewScanner(logs)

		for s.Scan() {
			record := k.parse(s.Text())
			record.Metadata["pod"] = pod
			records = append(records, record)
		}
	}

	// sort the records
	sort.Slice(records, func(i, j int) bool { return records[i].Timestamp.Before(records[j].Timestamp) })

	return records, nil
}

func (k *klog) Write(l log.Record) error {
	return write(l)
}

func (k *klog) Stream() (log.Stream, error) {
	// find the matching pods
	pods, err := k.getMatchingPods()
	if err != nil {
		return nil, err
	}

	stream := &kubeStream{
		stream: make(chan log.Record),
		stop:   make(chan bool),
	}

	// stream from the individual pods
	for _, pod := range pods {
		go k.podLogStream(pod, stream)
	}

	return stream, nil
}

// NewLog returns a configured Kubernetes logger
func NewLog(opts ...log.Option) log.Log {
	klog := &klog{}
	for _, o := range opts {
		o(&klog.Options)
	}

	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) > 0 {
		klog.client = client.NewClusterClient()
	} else {
		klog.client = client.NewLocalClient()
	}
	return klog
}
