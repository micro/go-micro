package sync

import (
	"fmt"
	"math"
	"time"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/sync/leader/etcd"
	"github.com/micro/go-micro/v2/sync/task"
	"github.com/micro/go-micro/v2/sync/task/local"
)

type syncCron struct {
	opts Options
}

func backoff(attempts int) time.Duration {
	if attempts == 0 {
		return time.Duration(0)
	}
	return time.Duration(math.Pow(10, float64(attempts))) * time.Millisecond
}

func (c *syncCron) Schedule(s task.Schedule, t task.Command) error {
	id := fmt.Sprintf("%s-%s", s.String(), t.String())

	go func() {
		// run the scheduler
		tc := s.Run()

		var i int

		for {
			// leader election
			e, err := c.opts.Leader.Elect(id)
			if err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Errorf("[cron] leader election error: %v", err)
				}
				time.Sleep(backoff(i))
				i++
				continue
			}

			i = 0
			r := e.Revoked()

			// execute the task
		Tick:
			for {
				select {
				// schedule tick
				case _, ok := <-tc:
					// ticked once
					if !ok {
						break Tick
					}

					if logger.V(logger.InfoLevel, logger.DefaultLogger) {
						logger.Infof("[cron] executing command %s", t.Name)
					}
					if err := c.opts.Task.Run(t); err != nil {
						if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
							logger.Errorf("[cron] error executing command %s: %v", t.Name, err)
						}
					}
				// leader revoked
				case <-r:
					break Tick
				}
			}

			// resign
			e.Resign()
		}
	}()

	return nil
}

func NewCron(opts ...Option) Cron {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Leader == nil {
		options.Leader = etcd.NewLeader()
	}

	if options.Task == nil {
		options.Task = local.NewTask()
	}

	return &syncCron{
		opts: options,
	}
}
