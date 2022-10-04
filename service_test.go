package micro

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"

	"go-micro.dev/v4/client"
	"go-micro.dev/v4/debug/handler"
	proto "go-micro.dev/v4/debug/proto"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go-micro.dev/v4/transport"
	"go-micro.dev/v4/util/test"
)

const (
	serviceName = "test.service"
)

func testService(ctx context.Context, tb testing.TB, wg *sync.WaitGroup, name string) Service {
	tb.Helper()

	// add self
	wg.Add(1)

	r := registry.NewMemoryRegistry(
		registry.Services(test.Data),
	)

	s := server.NewRPCServer(
		server.Name(name),
		server.Registry(r),
	)

	if err := s.Init(); err != nil {
		tb.Fatalf("[%s] Server init failed: %v", name, err)
	}

	// create service
	tb.Logf("Creating service: %v", name)
	srv := NewService(
		Server(s),
		Registry(r),
		Context(ctx),
		AfterStart(func() error {
			wg.Done()

			return nil
		}),
		// AfterStop(func() error {
		// 	wg.Done()
		//
		// 	return nil
		// }),
	)

	if err := RegisterHandler(srv.Server(), handler.NewHandler(srv.Client())); err != nil {
		tb.Fatalf("failed to register handler during initial service setup: %v", err)
	}

	return srv
}

func testCustomListenService(ctx context.Context, tb testing.TB, customListener net.Listener,
	wg *sync.WaitGroup, name string) Service {
	tb.Helper()

	// add self
	wg.Add(1)

	r := registry.NewMemoryRegistry(registry.Services(test.Data))

	s := server.NewRPCServer(
		server.Name(name),
		server.Registry(r),
		server.ListenOption(transport.NetListener(customListener)),
	)

	if err := s.Init(); err != nil {
		tb.Fatalf("[%s] Server init failed: %v", name, err)
	}

	// create service
	srv := NewService(
		Server(s),
		Registry(r),
		Context(ctx),
		AfterStart(func() error {
			wg.Done()
			return nil
		}),
		// AfterStop(func() error {
		// 	wg.Done()
		// 	return nil
		// }),
	)

	if err := RegisterHandler(srv.Server(), handler.NewHandler(srv.Client())); err != nil {
		tb.Fatal(err)
	}

	return srv
}

func testRequest(ctx context.Context, c client.Client, name string) error {
	// test call debug
	req := c.NewRequest(
		name,
		"Debug.Health",
		new(proto.HealthRequest),
	)

	rsp := new(proto.HealthResponse)

	// TODO: remvoe timeout
	if err := c.Call(ctx, req, rsp, client.WithRequestTimeout(30*time.Second)); err != nil {
		return err
	}

	if rsp.Status != "ok" {
		return errors.New("service response: " + rsp.Status)
	}

	return nil
}

func benchmarkCustomListenService(b *testing.B, n int, name string) {
	b.Helper()

	// Stop the timer
	b.StopTimer()

	customListen, err := net.Listen("tcp", server.DefaultAddress)
	if err != nil {
		b.Fatal(err)
	}

	// Waitgroup for server start
	var wg sync.WaitGroup

	// Cancellation context
	ctx, cancel := context.WithCancel(context.Background())

	// Create test server
	service := testCustomListenService(ctx, b, customListen, &wg, name)

	runBenchmark(b, service, &wg, cancel, name, n)
}

func benchmarkService(b *testing.B, n int, name string) {
	b.Helper()

	// stop the timer
	b.StopTimer()

	// waitgroup for server start
	var wg sync.WaitGroup

	// cancellation context
	ctx, cancel := context.WithCancel(context.Background())

	// create test server
	service := testService(ctx, b, &wg, name)

	runBenchmark(b, service, &wg, cancel, name, n)
}

func runBenchmark(b *testing.B, service Service,
	wg *sync.WaitGroup, cancel func(), name string, n int) {
	b.Helper()

	b.Logf("[%s] Starting benchmark test", name)

	// Receive error from routine on channel
	errChan := make(chan error, 1)

	// start the server
	done := make(chan struct{})
	go func() {
		b.Logf("[%s] Starting server for benchmark", name)
		if err := service.Run(); err != nil {
			errChan <- errors.Wrapf(err, "[%s] Error occurred during service.Run", name)
		}
		done <- struct{}{}
	}()

	// Benchmark routine
	sigTerm := make(chan struct{})
	go func() {
		// wait for service to start
		wg.Wait()
		b.Logf("[%s] Server started", name)

		// make a test call to warm the cache
		b.Logf("[%s] Warming cache", name)
		for i := 0; i < 10; i++ {
			if err := testRequest(context.Background(), service.Client(), name); err != nil {
				errChan <- errors.Wrapf(err, "[%s] Failure during cache warmup testRequest", name)
			}
		}

		b.Logf("[%s] Starting benchtest", name)
		b.StartTimer()

		defer func() {
			b.StopTimer()

			// shutdown service
			b.Logf("[%s] Shutting down", name)
			cancel()
			sigTerm <- struct{}{}
		}()

		// number of iterations
		for i := 0; i < b.N; i++ {
			// for concurrency
			for j := 0; j < n; j++ {
				wg.Add(1)

				go func(i, j int) {
					defer wg.Done()

					if err := testRequest(context.Background(), service.Client(), name); err != nil {
						b.Errorf("[%s] Request failed (%d/%d) (%d/%d)", name, i, b.N, j, n)
						errChan <- errors.Wrapf(err, "[%s] Error occurred during testRequest", name)
						return
					}
				}(i, j)
			}

			// wait for test completion
			wg.Wait()
		}
	}()

	defer func() {
		<-done
	}()

	// Catch any errors
	select {
	case err := <-errChan:
		b.Fatal(err)
	case <-sigTerm:
		b.Logf("[%s] Completed benchmark", name)
	}
}

// TestService tests running and calling a service.
func TestService(t *testing.T) {
	// waitgroup for server start
	var wg sync.WaitGroup

	// Cancellation context
	ctx, cancel := context.WithCancel(context.Background())

	// Start test server
	t.Log("[TestService] Running")
	service := testService(ctx, t, &wg, serviceName)

	// Receive error from routine on channel
	errChan := make(chan error, 1)

	// Start service
	go func() {
		t.Log("[TestService] Starting service")
		if err := service.Run(); err != nil {
			errChan <- err
		}
	}()

	go func() {
		// Wait for service to start
		wg.Wait()

		// Make a test call
		t.Log("[TestService] Making test request")
		if err := testRequest(context.Background(), service.Client(), serviceName); err != nil {
			errChan <- err
			return
		}

		// Shutdown the service
		t.Logf("[TestService] Shutting down")
		cancel()
	}()

	select {
	case err := <-errChan:
		t.Fatalf("[TestService] Error occurred during execution: %v", err)
	case <-ctx.Done():
	}
}

func BenchmarkService1(b *testing.B) {
	benchmarkService(b, 1, "benchmark.service.1")
}

func BenchmarkService8(b *testing.B) {
	benchmarkService(b, 8, "benchmark.service.8")
}

func BenchmarkService16(b *testing.B) {
	benchmarkService(b, 16, "benchmark.service.16")
}

func BenchmarkService32(b *testing.B) {
	benchmarkService(b, 32, "benchmark.service.32")
}

func BenchmarkService64(b *testing.B) {
	benchmarkService(b, 64, "benchmark.service.64")
}

func BenchmarkCustomListenService1(b *testing.B) {
	benchmarkCustomListenService(b, 1, "customlistener.service.1")
}
