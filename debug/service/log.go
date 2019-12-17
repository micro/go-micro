package service

import (
	"github.com/micro/go-micro/debug"
	"github.com/micro/go-micro/debug/log"
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
	for record := range stream {
		records = append(records, record)
	}
	return records
}

// There is no write support
func (s *serviceLog) Write(r log.Record) {
	return
}

// Stream log records
func (s *serviceLog) Stream() (<-chan log.Record, chan bool) {
	stop := make(chan bool)
	stream, err := s.Client.Log(log.Stream(true))
	if err != nil {
		// return a closed stream
		deadStream := make(chan log.Record)
		close(deadStream)
		return deadStream, stop
	}

	newStream := make(chan log.Record, 128)

	go func() {
		for {
			select {
			case rec := <-stream:
				newStream <- rec
			case <-stop:
				return
			}
		}
	}()

	return newStream, stop
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

	return &serviceLog{
		Client: NewClient(name),
	}
}
