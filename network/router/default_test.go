package router

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/registry/memory"
)

func routerTestSetup() Router {
	r := memory.NewRegistry()
	return newRouter(Registry(r))
}

func TestRouterStartStop(t *testing.T) {
	r := routerTestSetup()

	if err := r.Start(); err != nil {
		t.Errorf("failed to start router: %v", err)
	}

	_, err := r.Advertise()
	if err != nil {
		t.Errorf("failed to start advertising: %v", err)
	}

	if err := r.Stop(); err != nil {
		t.Errorf("failed to stop router: %v", err)
	}
	if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
		t.Logf("TestRouterStartStop STOPPED")
	}
}

func TestRouterAdvertise(t *testing.T) {
	r := routerTestSetup()

	// lower the advertise interval
	AdvertiseEventsTick = 500 * time.Millisecond

	if err := r.Start(); err != nil {
		t.Errorf("failed to start router: %v", err)
	}

	ch, err := r.Advertise()
	if err != nil {
		t.Errorf("failed to start advertising: %v", err)
	}

	// receive announce event
	ann := <-ch
	if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
		t.Logf("received announce advert: %v", ann)
	}

	// Generate random unique routes
	nrRoutes := 5
	routes := make([]Route, nrRoutes)
	route := Route{
		Service: "dest.svc",
		Address: "dest.addr",
		Gateway: "dest.gw",
		Network: "dest.network",
		Router:  "src.router",
		Link:    "local",
		Metric:  10,
	}

	for i := 0; i < nrRoutes; i++ {
		testRoute := route
		testRoute.Service = fmt.Sprintf("%s-%d", route.Service, i)
		routes[i] = testRoute
	}

	var advertErr error

	createDone := make(chan bool)
	errChan := make(chan error)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		defer close(createDone)
		for _, route := range routes {
			if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
				t.Logf("Creating route %v", route)
			}
			if err := r.Table().Create(route); err != nil {
				if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
					t.Logf("Failed to create route: %v", err)
				}
				errChan <- err
				return
			}
		}
	}()

	var adverts int
	readDone := make(chan bool)

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			readDone <- true
		}()
		for advert := range ch {
			select {
			case advertErr = <-errChan:
				t.Errorf("failed advertising events: %v", advertErr)
			default:
				// do nothing for now
				if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
					t.Logf("Router advert received: %v", advert)
				}
				adverts += len(advert.Events)
			}
			return
		}
	}()

	// done adding routes to routing table
	<-createDone
	// done reading adverts from the routing table
	<-readDone

	if adverts != nrRoutes {
		t.Errorf("Expected %d adverts, received: %d", nrRoutes, adverts)
	}

	wg.Wait()

	if err := r.Stop(); err != nil {
		t.Errorf("failed to stop router: %v", err)
	}
}
