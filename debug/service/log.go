package service

import (
	"context"
	"fmt"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/debug/log"
	pb "github.com/micro/go-micro/debug/service/proto"
)

type serviceLog struct {
	Client *debugClient
}

// Read reads log entries from the logger
func (s *serviceLog) Read(opts ...log.ReadOption) []log.Record {
	// TODO: parse opts
	stream, err := s.Client.Log(opts...)
	if err != nil {
		return nil
	}
	// stream the records until nothing is left
	var records []log.Record
	for _, record := range stream {
		records = append(records, record)
	}
	return records
}

// There is no write support
func (s *serviceLog) Write(r log.Record) {
	return
}

// Stream log records
func (s *serviceLog) Stream(ch chan bool) (<-chan log.Record, chan bool) {
	stop := make(chan bool)
	stream, err := s.Client.Log(log.Stream(true))
	if err != nil {
		// return a closed stream
		stream = make(chan log.Record)
		close(stream)
		return stream, stop
	}

	// stream the records until nothing is left
	go func() {
		var records []log.Record
		for _, record := range stream {
			select {
			case stream <- record:
			case <-stop:
				return
			}
		}
	}()

	// return the stream
	return stream, stop
}

// NewLog returns a new log interface
func NewLog(opts ...log.Option) log.Log {
	var options log.Options
	for _, o := range opts {
		o(&options)
	}

	name := options.Name

	// set the default name
	if len(name) == 0 {
		name = debug.DefaultName
	}

	return serviceLog{
		Client: newDebugClient(name),
	}
}
