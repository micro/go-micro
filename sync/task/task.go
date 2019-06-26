// Package task provides an interface for distributed jobs
package task

import (
	"context"
	"fmt"
	"time"
)

// Task represents a distributed task
type Task interface {
	// Run runs a command immediately until completion
	Run(Command) error
	// Status provides status of last execution
	Status() string
}

// Command to be executed
type Command struct {
	Name string
	Func func() error
}

// Schedule represents a time or interval at which a task should run
type Schedule struct {
	// When to start the schedule. Zero time means immediately
	Time time.Time
	// Non zero interval dictates an ongoing schedule
	Interval time.Duration
}

type Options struct {
	// Pool size for workers
	Pool int
	// Alternative options
	Context context.Context
}

type Option func(o *Options)

func (c Command) Execute() error {
	return c.Func()
}

func (c Command) String() string {
	return c.Name
}

func (s Schedule) Run() <-chan time.Time {
	d := s.Time.Sub(time.Now())

	ch := make(chan time.Time, 1)

	go func() {
		// wait for start time
		<-time.After(d)

		// zero interval
		if s.Interval == time.Duration(0) {
			ch <- time.Now()
			close(ch)
			return
		}

		// start ticker
		for t := range time.Tick(s.Interval) {
			ch <- t
		}
	}()

	return ch
}

func (s Schedule) String() string {
	return fmt.Sprintf("%d-%d", s.Time.Unix(), s.Interval)
}

// WithPool sets the pool size for concurrent work
func WithPool(i int) Option {
	return func(o *Options) {
		o.Pool = i
	}
}
