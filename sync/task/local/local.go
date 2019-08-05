// Package local provides a local task runner
package local

import (
	"fmt"
	"sync"

	"github.com/micro/go-micro/sync/task"
)

type localTask struct {
	opts   task.Options
	mtx    sync.RWMutex
	status string
}

func (l *localTask) Run(t task.Command) error {
	ch := make(chan error, l.opts.Pool)

	for i := 0; i < l.opts.Pool; i++ {
		go func() {
			ch <- t.Execute()
		}()
	}

	var err error

	for i := 0; i < l.opts.Pool; i++ {
		er := <-ch
		if err != nil {
			err = er
			l.mtx.Lock()
			l.status = fmt.Sprintf("command [%s] status: %s", t.Name, err.Error())
			l.mtx.Unlock()
		}
	}

	close(ch)
	return err
}

func (l *localTask) Status() string {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.status
}

func NewTask(opts ...task.Option) task.Task {
	var options task.Options
	for _, o := range opts {
		o(&options)
	}
	if options.Pool == 0 {
		options.Pool = 1
	}
	return &localTask{
		opts: options,
	}
}
