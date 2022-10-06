package micro

import (
	"context"
	"net"
	"sync"
	"testing"

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

type ServiceConfig struct {
	Name      string
	Registry  registry.Registry
	Transport transport.Transport
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

func BenchmarkService128(b *testing.B) {
	benchmarkService(b, 128, "benchmark.service.128")
}

func BenchmarkCustomListenService1(b *testing.B) {
	benchmarkCustomListenService(b, 1, "customlistener.service.1")
}

// TestService tests running and calling a service.
func TestService(t *testing.T) {
	reqCount := 1000

	// Start test server
	t.Log("[TestService] Running")
	service, ch, cancel := newTestService(t, serviceName)

	// Receive error from routine on channel
	errChan := make(chan error, 1)

	// Start service
	sigTerm := make(chan struct{})
	go func() {
		t.Log("[TestService] Starting service")
		if err := service.Run(); err != nil {
			errChan <- err
		}
		sigTerm <- struct{}{}
	}()

	go func() {
		// Wait for service to start
		<-ch

		// First make sequential requests
		for i := 0; i < reqCount; i++ {
			if err := testRequest(context.Background(), service.Client(), serviceName); err != nil {
				errChan <- err
				return
			}
		}

		// Second make parallel requests
		wg := sync.WaitGroup{}
		for i := 0; i < reqCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				if err := testRequest(context.Background(), service.Client(), serviceName); err != nil {
					errChan <- err
					return
				}
			}()
		}
		wg.Wait()

		// Shutdown the service
		t.Logf("[TestService] Shutting down")
		cancel()
	}()

	select {
	case err := <-errChan:
		t.Fatalf("[TestService] Error occurred during execution: %v", err)
	case <-sigTerm:
	}
}

func newTestService(tb testing.TB, name string) (Service, chan struct{}, context.CancelFunc) {
	tb.Helper()

	r := registry.NewMemoryRegistry(
		registry.Services(test.Data),
	)

	t := transport.NewHTTPTransport()
	c := client.NewClient(client.Transport(t))

	s := server.NewRPCServer(
		server.Name(name),
		server.Registry(r),
		server.Transport(t),
	)

	if err := s.Init(); err != nil {
		tb.Fatalf("[%s] Server init failed: %v", name, err)
	}

	// Channel to signal service has started
	ch := make(chan struct{})

	// Context with cancel to stop the service
	ctx, cancel := context.WithCancel(context.Background())

	// create service
	tb.Logf("Creating service: %v", name)
	srv := NewService(
		Server(s),
		Client(c),
		Registry(r),
		Context(ctx),
		AfterStart(func() error {
			ch <- struct{}{}

			return nil
		}),
	)

	if err := RegisterHandler(srv.Server(), handler.NewHandler(srv.Client())); err != nil {
		tb.Fatalf("failed to register handler during initial service setup: %v", err)
	}

	return srv, ch, cancel
}

func testCustomListenService(tb testing.TB, customListener net.Listener, name string) (Service, chan struct{}, context.CancelFunc) {
	tb.Helper()

	r := registry.NewMemoryRegistry(registry.Services(test.Data))

	s := server.NewRPCServer(
		server.Name(name),
		server.Registry(r),
		server.ListenOption(transport.NetListener(customListener)),
	)

	if err := s.Init(); err != nil {
		tb.Fatalf("[%s] Server init failed: %v", name, err)
	}

	// Channel to signal service has started
	ch := make(chan struct{})

	// Context with cancel to stop the service
	ctx, cancel := context.WithCancel(context.Background())

	// create service
	srv := NewService(
		Server(s),
		Registry(r),
		Context(ctx),
		AfterStart(func() error {
			ch <- struct{}{}

			return nil
		}),
	)

	if err := RegisterHandler(srv.Server(), handler.NewHandler(srv.Client())); err != nil {
		tb.Fatal(err)
	}

	return srv, ch, cancel
}

func testRequest(ctx context.Context, c client.Client, name string) error {
	// test call debug
	req := c.NewRequest(
		name,
		"Debug.Health",
		new(proto.HealthRequest),
	)

	rsp := new(proto.HealthResponse)

	if err := c.Call(ctx, req, rsp); // client.WithConnClose(),
	err != nil {
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

	// Create test server
	service, ch, cancel := testCustomListenService(b, customListen, name)

	RunBenchmark(b, service, ch, cancel, name, n)
}

func benchmarkService(b *testing.B, n int, name string) {
	b.Helper()

	// Stop the timer
	b.StopTimer()

	// Create test server
	service, ch, cancel := newTestService(b, name)

	RunBenchmark(b, service, ch, cancel, name, n)
}

func RunBenchmark(b *testing.B, service Service, ch chan struct{}, cancel func(), name string, n int) {
	b.Helper()

	b.Logf("[%s] Starting benchmark test", name)

	wg := sync.WaitGroup{}

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
		<-ch
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

		// Number of iterations
		for i := 0; i < b.N; i++ {
			// For concurrency
			for j := 0; j < n; j++ {
				wg.Add(1)

				go func(i, j int) {
					defer wg.Done()

					if err := testRequest(context.Background(), service.Client(), name); err != nil {
						b.Errorf("[%s] Request failed (%d/%d) (%d/%d)", name, i+1, b.N, j+1, n)
						errChan <- errors.Wrapf(err, "[%s] Error occurred during testRequest", name)
						return
					}
				}(i, j)
			}

			// Wait for test completion
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
