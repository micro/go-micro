// Package broker provides a distributed task manager built on the micro broker
package broker

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/sync/task"
)

type brokerKey struct{}

// Task is a broker task
type Task struct {
	// a micro broker
	Broker broker.Broker
	// Options
	Options task.Options

	mtx    sync.RWMutex
	status string
}

func returnError(err error, ch chan error) {
	select {
	case ch <- err:
	default:
	}
}

func (t *Task) Run(c task.Command) error {
	// connect
	t.Broker.Connect()
	// unique id for this runner
	id := uuid.New().String()
	// topic of the command
	topic := fmt.Sprintf("task.%s", c.Name)

	// global error
	errCh := make(chan error, t.Options.Pool)

	// subscribe for distributed work
	workFn := func(p broker.Event) error {
		msg := p.Message()

		// get command name
		command := msg.Header["Command"]

		// check the command is what we expect
		if command != c.Name {
			returnError(errors.New("received unknown command: "+command), errCh)
			return nil
		}

		// new task created
		switch msg.Header["Status"] {
		case "start":
			// artificially delay start of processing
			time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(100)))

			// execute the function
			err := c.Func()

			status := "done"
			errors := ""

			if err != nil {
				status = "error"
				errors = err.Error()
			}

			// create response
			msg := &broker.Message{
				Header: map[string]string{
					"Command":   c.Name,
					"Error":     errors,
					"Id":        id,
					"Status":    status,
					"Timestamp": fmt.Sprintf("%d", time.Now().Unix()),
				},
				// Body is nil, may be used in future
			}

			// publish end of task
			if err := t.Broker.Publish(topic, msg); err != nil {
				returnError(err, errCh)
			}
		}

		return nil
	}

	// subscribe for the pool size
	for i := 0; i < t.Options.Pool; i++ {
		// subscribe to work
		subWork, err := t.Broker.Subscribe(topic, workFn, broker.Queue(fmt.Sprintf("work.%d", i)))
		if err != nil {
			return err
		}

		// unsubscribe on completion
		defer subWork.Unsubscribe()
	}

	// subscribe to all status messages
	subStatus, err := t.Broker.Subscribe(topic, func(p broker.Event) error {
		msg := p.Message()

		// get command name
		command := msg.Header["Command"]

		// check the command is what we expect
		if command != c.Name {
			return nil
		}

		// check task status
		switch msg.Header["Status"] {
		// task is complete
		case "done":
			errCh <- nil
		// someone failed
		case "error":
			returnError(errors.New(msg.Header["Error"]), errCh)
		}

		return nil
	})
	if err != nil {
		return err
	}
	defer subStatus.Unsubscribe()

	// a new task
	msg := &broker.Message{
		Header: map[string]string{
			"Command":   c.Name,
			"Id":        id,
			"Status":    "start",
			"Timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	// artificially delay the start of the task
	time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(100)))

	// publish the task
	if err := t.Broker.Publish(topic, msg); err != nil {
		return err
	}

	var gerrors []string

	// wait for all responses
	for i := 0; i < t.Options.Pool; i++ {
		// check errors
		err := <-errCh

		// append to errors
		if err != nil {
			gerrors = append(gerrors, err.Error())
		}
	}

	// return the errors
	if len(gerrors) > 0 {
		return errors.New("errors: " + strings.Join(gerrors, "\n"))
	}

	return nil
}

func (t *Task) Status() string {
	t.mtx.RLock()
	defer t.mtx.RUnlock()
	return t.status
}

// Broker sets the micro broker
func WithBroker(b broker.Broker) task.Option {
	return func(o *task.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, brokerKey{}, b)
	}
}

// NewTask returns a new broker task
func NewTask(opts ...task.Option) task.Task {
	options := task.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Pool == 0 {
		options.Pool = 1
	}

	b, ok := options.Context.Value(brokerKey{}).(broker.Broker)
	if !ok {
		b = broker.DefaultBroker
	}

	return &Task{
		Broker:  b,
		Options: options,
	}
}
