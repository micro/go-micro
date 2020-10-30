package registry

import (
	"testing"

	"github.com/asim/nitro/v3/router"
)

func testSetup() (*table, router.Route) {
	table := newTable()

	route := router.Route{
		Service: "dest.svc",
		Address: "dest.addr",
		Gateway: "dest.gw",
		Network: "dest.network",
		Router:  "src.router",
		Link:    "det.link",
		Metric:  10,
	}

	return table, route
}

func TestCreate(t *testing.T) {
	table, route := testSetup()

	if err := table.Create(route); err != nil {
		t.Fatalf("error adding route: %s", err)
	}

	// adds new route for the original destination
	route.Gateway = "dest.gw2"

	if err := table.Create(route); err != nil {
		t.Fatalf("error adding route: %s", err)
	}

	// adding the same route under Insert policy must error
	if err := table.Create(route); err != router.ErrDuplicateRoute {
		t.Fatalf("error adding route. Expected error: %s, found: %s", router.ErrDuplicateRoute, err)
	}
}

func TestDelete(t *testing.T) {
	table, route := testSetup()

	if err := table.Create(route); err != nil {
		t.Fatalf("error adding route: %s", err)
	}

	// should fail to delete non-existant route
	prevSvc := route.Service
	route.Service = "randDest"

	if err := table.Delete(route); err != router.ErrRouteNotFound {
		t.Fatalf("error deleting route. Expected: %s, found: %s", router.ErrRouteNotFound, err)
	}

	// we should be able to delete the existing route
	route.Service = prevSvc

	if err := table.Delete(route); err != nil {
		t.Fatalf("error deleting route: %s", err)
	}
}

func TestUpdate(t *testing.T) {
	table, route := testSetup()

	if err := table.Create(route); err != nil {
		t.Fatalf("error adding route: %s", err)
	}

	// change the metric of the original route
	route.Metric = 200

	if err := table.Update(route); err != nil {
		t.Fatalf("error updating route: %s", err)
	}

	// this should add a new route
	route.Service = "rand.dest"

	if err := table.Update(route); err != nil {
		t.Fatalf("error updating route: %s", err)
	}
}

func TestList(t *testing.T) {
	table, route := testSetup()

	svc := []string{"one.svc", "two.svc", "three.svc"}

	for i := 0; i < len(svc); i++ {
		route.Service = svc[i]
		if err := table.Create(route); err != nil {
			t.Fatalf("error adding route: %s", err)
		}
	}

	routes, err := table.Read()
	if err != nil {
		t.Fatalf("error listing routes: %s", err)
	}

	if len(routes) != len(svc) {
		t.Fatalf("incorrect number of routes listed. Expected: %d, found: %d", len(svc), len(routes))
	}
}

func TestQuery(t *testing.T) {
	table, route := testSetup()

	if err := table.Create(route); err != nil {
		t.Fatalf("error adding route: %s", err)
	}

	rt, err := table.Read(router.ReadService(route.Service))
	if err != nil {
		t.Fatal("Expected a route got err", err)
	}

	if len(rt) != 1 {
		t.Fatalf("Expected one route got %d", len(rt))
	}

	if rt[0].Hash() != route.Hash() {
		t.Fatal("Mismatched routes received")
	}
}
