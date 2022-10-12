// Package tests implements a testing framwork, and provides default tests.
package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"

	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/debug/handler"

	pb "go-micro.dev/v4/debug/proto"
)

var (
	// ErrNoTests returns no test params are set.
	ErrNoTests = errors.New("No tests to run, all values set to 0")
)

type parTest func(name string, c client.Client, p, s int, errChan chan error)
type testFunc func(name string, c client.Client, errChan chan error)

// ServiceTestConfig allows you to easilty test a service configuration by
// running predefined tests against your custom service. You only need to
// provide a function to create the service, and how many of which test you
// want to run.
//
// The default tests provided are Sequential Call requests running in parallel
// routines. And bi-directional stream requests, running in parallel routines.
//
// You can provide an array of parallel routines to run for the request and
// stream tests. They will be run as matrix tests, so with each possible combination.
// Thus, in total (p * seq) + (p * streams) tests will be run.
type ServiceTestConfig struct {
	// Service name to use for the tests
	Name string
	// NewService function will be called to setup the new service.
	// It takes in a list of options, which by default will Context and an
	// AfterStart with channel to signal when the service has been started.
	NewService func(name string, opts ...micro.Option) (micro.Service, error)
	// The number of prallell routines to use for the tests.
	Parallel []int
	// The number of sequential requests to send per parallel process.
	Sequential []int
	// The nummber of streaming messages to send over the stream per routine.
	Streams []int
}

// Run will start the benchmark tests.
func (stc *ServiceTestConfig) Run(b *testing.B) {
	if err := stc.validate(); err != nil {
		b.Fatal("Failed to validate config", err)
	}

	// Run routines with sequential requests
	stc.prepBench(b, "req", stc.runParSeqTest, stc.Sequential)

	// Run routines with streams
	stc.prepBench(b, "streams", stc.runParStreamTest, stc.Streams)
}

// prepBench will prepare the benmark by setting the right parameters,
// and invoking the test.
func (stc *ServiceTestConfig) prepBench(b *testing.B, tName string, test parTest, seq []int) {
	par := stc.Parallel

	// No requests needed
	if len(seq) == 0 || seq[0] == 0 {
		return
	}

	for _, parallel := range par {
		for _, sequential := range seq {
			// Create the service name for the test
			name := fmt.Sprintf("%s.%dp-%d%s", stc.Name, parallel, sequential, tName)

			// Run test with parallel routines making each sequential requests
			test := func(name string, c client.Client, errChan chan error) {
				test(name, c, parallel, sequential, errChan)
			}

			benchmark := func(b *testing.B) {
				stc.runBench(b, name, test)
			}

			b.Logf("----------- STARTING TEST %s -----------", name)

			// Run test, return if it fails
			if !b.Run(name, benchmark) {
				return
			}
		}
	}
}

// runParSeqTest will make s sequential requests in p parallel routines.
func (stc *ServiceTestConfig) runParSeqTest(name string, c client.Client, p, s int, errChan chan error) {
	testParallel(p, func() {
		// Make serial requests
		for z := 0; z < s; z++ {
			if err := testRequest(context.Background(), c, name); err != nil {
				errChan <- errors.Wrapf(err, "[%s] Request failed during testRequest", name)
				return
			}
		}
	})
}

// runParStreamTest will start streaming, and send s messages parallel in p routines.
func (stc *ServiceTestConfig) runParStreamTest(name string, c client.Client, p, s int, errChan chan error) {
	testParallel(p, func() {
		// Create a client service
		srv := pb.NewDebugService(name, c)

		// Establish a connection to server over which we start streaming
		bus, err := srv.MessageBus(context.Background())
		if err != nil {
			errChan <- errors.Wrap(err, "failed to connect to message bus")
			return
		}

		// Start streaming requests
		for z := 0; z < s; z++ {
			if err := bus.Send(&pb.BusMsg{Msg: "Hack the world!"}); err != nil {
				errChan <- errors.Wrap(err, "failed to send to  stream")
				return
			}

			msg, err := bus.Recv()
			if err != nil {
				errChan <- errors.Wrap(err, "failed to receive message from stream")
				return
			}

			expected := "Request received!"
			if msg.Msg != expected {
				errChan <- fmt.Errorf("stream returned unexpected mesage. Expected '%s', but got '%s'", expected, msg.Msg)
				return
			}
		}
	})
}

// validate will make sure the provided test parameters are a legal combination.
func (stc *ServiceTestConfig) validate() error {
	lp, lseq, lstr := len(stc.Parallel), len(stc.Sequential), len(stc.Streams)

	if lp == 0 || (lseq == 0 && lstr == 0) {
		return ErrNoTests
	}

	return nil
}

// runBench will create a service with the provided stc.NewService function,
// and run a benchmark on the test function.
func (stc *ServiceTestConfig) runBench(b *testing.B, name string, test testFunc) {
	b.StopTimer()

	// Channel to signal service has started
	started := make(chan struct{})

	// Context with cancel to stop the service
	ctx, cancel := context.WithCancel(context.Background())

	opts := []micro.Option{
		micro.Context(ctx),
		micro.AfterStart(func() error {
			started <- struct{}{}
			return nil
		}),
	}

	// Create a new service per test
	service, err := stc.NewService(name, opts...)
	if err != nil {
		b.Fatalf("failed to create service: %v", err)
	}

	// Register handler
	if err := pb.RegisterDebugHandler(service.Server(), handler.NewHandler(service.Client())); err != nil {
		b.Fatalf("failed to register handler during initial service setup: %v", err)
	}

	RunBenchmark(b, name, service, test, cancel, started)
}

// RunBenchmark will run benchmarks on a provided service.
//
// A test function can be provided that will be fun b.N times.
// make sure to call b.Stoptimer() BEFORE calling this function, as you will
// probably want to call it before even creating your service.
func RunBenchmark(b *testing.B, name string, service micro.Service, test testFunc,
	cancel context.CancelFunc, started chan struct{}) {
	// Receive errors from routines on this channel
	errChan := make(chan error, 1)

	// Receive singal after service has shutdown
	done := make(chan struct{})

	// Start the server
	go func() {
		b.Logf("[%s] Starting server for benchmark", name)

		if err := service.Run(); err != nil {
			errChan <- errors.Wrapf(err, "[%s] Error occurred during service.Run", name)
		}
		done <- struct{}{}
	}()

	sigTerm := make(chan struct{})

	// Benchmark routine
	go func() {
		// Wait for service to start
		<-started

		// Give the registry more time to setup
		time.Sleep(time.Second)

		b.Logf("[%s] Server started", name)

		// Make a test call to warm the cache
		for i := 0; i < 10; i++ {
			if err := testRequest(context.Background(), service.Client(), name); err != nil {
				errChan <- errors.Wrapf(err, "[%s] Failure during cache warmup testRequest", name)
			}
		}

		defer func() {
			b.StopTimer()

			// Shutdown service
			b.Logf("[%s] Shutting down", name)
			cancel()

			// Wait for service to be fully stopped
			<-done
			sigTerm <- struct{}{}
		}()

		// Start benchmark
		b.Logf("[%s] Starting benchtest", name)
		b.StartTimer()

		// Number of iterations
		for i := 0; i < b.N; i++ {
			test(name, service.Client(), errChan)
		}
	}()

	// Wait for completion or catch any errors
	select {
	case err := <-errChan:
		b.Fatal(err)
	case <-sigTerm:
		b.Logf("[%s] Completed benchmark", name)
	}
}

// testParallel will run the test function in p parallel routines.
func testParallel(p int, test func()) {
	// Waitgroup to wait for requests to finish
	wg := sync.WaitGroup{}

	// For concurrency
	for j := 0; j < p; j++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			test()
		}()
	}

	// Wait for test completion
	wg.Wait()
}

// testRequest sends one test request.
// It calls the Debug.Health endpoint, and validates if the response returned
// contains the expected message.
func testRequest(ctx context.Context, c client.Client, name string) error {
	req := c.NewRequest(
		name,
		"Debug.Health",
		new(pb.HealthRequest),
	)

	rsp := new(pb.HealthResponse)

	if err := c.Call(ctx, req, rsp); err != nil {
		return err
	}

	if rsp.Status != "ok" {
		return errors.New("service response: " + rsp.Status)
	}

	return nil
}
