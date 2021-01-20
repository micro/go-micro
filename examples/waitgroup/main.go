package main

import (
	"fmt"
	"sync"

	"context"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/server"
)

// waitgroup is a handler wrapper which adds a handler to a sync.WaitGroup
func waitgroup(wg *sync.WaitGroup) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			wg.Add(1)
			defer wg.Done()
			return h(ctx, req, rsp)
		}
	}
}

func main() {
	var wg sync.WaitGroup

	service := micro.NewService(
		// wrap handlers with waitgroup wrapper
		micro.WrapHandler(waitgroup(&wg)),
		// waits for the waitgroup once stopped
		micro.AfterStop(func() error {
			// wait for handlers to finish
			wg.Wait()
			return nil
		}),
	)

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
