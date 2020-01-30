package service

import (
	"time"

	"github.com/micro/go-micro/v2/debug"
	"github.com/micro/go-micro/v2/debug/log"
)

type serviceLog struct {
	Client *debugClient
}

// Read reads log entries from the logger
func (s *serviceLog) Read(opts ...log.ReadOption) ([]log.Record, error) {
	var options log.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	stream, err := s.Client.Log(options.Since, options.Count, false)
	if err != nil {
		return nil, err
	}
	defer stream.Stop()

	// stream the records until nothing is left
	var records []log.Record

	for record := range stream.Chan() {
		records = append(records, record)
	}

	return records, nil
}

// There is no write support
func (s *serviceLog) Write(r log.Record) error {
	return nil
}

// Stream log records
func (s *serviceLog) Stream() (log.Stream, error) {
	return s.Client.Log(time.Time{}, 0, true)
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
