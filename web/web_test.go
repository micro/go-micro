package web_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/web"
)

func TestWeb(t *testing.T) {
	for i := 0; i < 10; i++ {
		testFunc()
	}
}

func testFunc() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
	defer cancel()

	service := micro.NewService(
		micro.Name("test"),
		micro.Context(ctx),
		micro.HandleSignal(false),
		micro.Flags(
			&cli.StringFlag{
				Name: "test.timeout",
			},
			&cli.BoolFlag{
				Name: "test.v",
			},
			&cli.StringFlag{
				Name: "test.run",
			},
			&cli.StringFlag{
				Name: "test.testlogfile",
			},
		),
	)
	w := web.NewService(
		web.MicroService(service),
		web.Context(ctx),
		web.HandleSignal(false),
	)
	// s.Init()
	// w.Init()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := service.Run()
		if err != nil {
			logger.Logf(logger.ErrorLevel, "micro run error: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := w.Run()
		if err != nil {
			logger.Logf(logger.ErrorLevel, "web run error: %v", err)
		}
	}()

	wg.Wait()
}
