// Package kubernetes is a logger implementing (github.com/micro/go-micro/debug/log).Log
package kubernetes

import (
	"github.com/micro/go-micro/debug/log"
)

type klog struct{}

func (k *klog) Read(...log.ReadOption) []log.Record { return nil }

func (k *klog) Write(l log.Record) {
	write(l)
}

func (k *klog) Stream() (<-chan log.Record, chan bool) {
	c, s := make(chan log.Record), make(chan bool)
	go close(c)
	return c, s
}

// New returns a configured Kubernetes logger
func New() log.Log {
	return &klog{}
}
