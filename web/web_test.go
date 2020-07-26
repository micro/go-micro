package web_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/web"
)

func TestWeb(t *testing.T) {
	for i := 0; i < 3; i++ {
		fmt.Println("Test nr", i)
		testFunc()
	}
}

func testFunc() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
	defer cancel()

	w := web.NewService(
		web.Context(ctx),
		web.HandleSignal(false),
	)
	//s.Init()
	//w.Init()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := w.Run()
		if err != nil {
			logger.Errorf("web run error: %v", err)
		}
	}()

	wg.Wait()
}
