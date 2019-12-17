// Package kubernetes is a logger implementing (github.com/micro/go-micro/debug/log).Log
package kubernetes

import (
	"errors"

	"github.com/micro/go-micro/debug/log"
)

type klog struct{}

func (k *klog) Read(...log.ReadOption) ([]log.Record, error) {
	return nil, errors.New("not implemented")
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
func New() log.Log {
	return &klog{}
}
