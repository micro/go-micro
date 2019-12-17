// Package kubernetes is a logger implementing (github.com/micro/go-micro/debug/log).Log
package kubernetes

import (
	"github.com/micro/go-micro/debug/log"
)

type klog struct{}

func (k *klog) Read(...log.ReadOption) []log.Record { return nil }

func (k *klog) Write(log.Record) {}

func (k *klog) Stream(stop chan bool) <-chan log.Record {
	c := make(chan log.Record)
	go close(c)
	return c
}

// New returns a configured Kubernetes logger
func New() log.Log {
	return &klog{}
}
