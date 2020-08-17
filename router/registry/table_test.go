package registry

import (
	"fmt"
	"testing"

	"github.com/micro/go-micro/v3/router"
)

func testSetup() (*table, router.Route) {
	routr := NewRouter().(*rtr)
	table := newTable(routr.lookup)

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

	routes, err := table.List()
	if err != nil {
		t.Fatalf("error listing routes: %s", err)
	}

	if len(routes) != len(svc) {
		t.Fatalf("incorrect number of routes listed. Expected: %d, found: %d", len(svc), len(routes))
	}
}

func TestQuery(t *testing.T) {
	table, route := testSetup()

	svc := []string{"svc1", "svc2", "svc3", "svc1"}
	net := []string{"net1", "net2", "net1", "net3"}
	gw := []string{"gw1", "gw2", "gw3", "gw3"}
	rtr := []string{"rtr1", "rt2", "rt3", "rtr3"}

	for i := 0; i < len(svc); i++ {
		route.Service = svc[i]
		route.Network = net[i]
		route.Gateway = gw[i]
		route.Router = rtr[i]
		route.Link = router.DefaultLink

		if err := table.Create(route); err != nil {
			t.Fatalf("error adding route: %s", err)
		}
	}

	// return all routes
	routes, err := table.Query()
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	} else if len(routes) == 0 {
		t.Fatalf("error looking up routes: not found")
	}

	// query routes particular network
	network := "net1"

	routes, err = table.Query(router.QueryNetwork(network))
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) != 2 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 2, len(routes))
	}

	for _, route := range routes {
		if route.Network != network {
			t.Fatalf("incorrect route returned. Expected network: %s, found: %s", network, route.Network)
		}
	}

	// query routes for particular gateway
	gateway := "gw1"

	routes, err = table.Query(router.QueryGateway(gateway))
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) != 1 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 1, len(routes))
	}

	if routes[0].Gateway != gateway {
		t.Fatalf("incorrect route returned. Expected gateway: %s, found: %s", gateway, routes[0].Gateway)
	}

	// query routes for particular router
	rt := "rtr1"

	routes, err = table.Query(router.QueryRouter(rt))
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) != 1 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 1, len(routes))
	}

	if routes[0].Router != rt {
		t.Fatalf("incorrect route returned. Expected router: %s, found: %s", rt, routes[0].Router)
	}

	// query particular gateway and network
	query := []router.QueryOption{
		router.QueryGateway(gateway),
		router.QueryNetwork(network),
		router.QueryRouter(rt),
	}

	routes, err = table.Query(query...)
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) != 1 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 1, len(routes))
	}

	if routes[0].Gateway != gateway {
		t.Fatalf("incorrect route returned. Expected gateway: %s, found: %s", gateway, routes[0].Gateway)
	}

	if routes[0].Network != network {
		t.Fatalf("incorrect network returned. Expected network: %s, found: %s", network, routes[0].Network)
	}

	if routes[0].Router != rt {
		t.Fatalf("incorrect route returned. Expected router: %s, found: %s", rt, routes[0].Router)
	}

	// non-existen route query
	routes, err = table.Query(router.QueryService("foobar"))
	if err != router.ErrRouteNotFound {
		t.Fatalf("error looking up routes. Expected: %s, found: %s", router.ErrRouteNotFound, err)
	}

	if len(routes) != 0 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 0, len(routes))
	}

	// query NO routes
	query = []router.QueryOption{
		router.QueryGateway(gateway),
		router.QueryNetwork(network),
		router.QueryLink("network"),
	}

	routes, err = table.Query(query...)
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) > 0 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 0, len(routes))
	}

	// insert local routes to query
	for i := 0; i < 2; i++ {
		route.Link = "foobar"
		route.Address = fmt.Sprintf("local.route.address-%d", i)
		if err := table.Create(route); err != nil {
			t.Fatalf("error adding route: %s", err)
		}
	}

	// query local routes
	query = []router.QueryOption{
		router.QueryGateway("*"),
		router.QueryNetwork("*"),
		router.QueryLink("foobar"),
	}

	routes, err = table.Query(query...)
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) != 2 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 2, len(routes))
	}

	// add two different routes for svcX with different metric
	for i := 0; i < 2; i++ {
		route.Service = "svcX"
		route.Address = fmt.Sprintf("svcX.route.address-%d", i)
		route.Metric = int64(100 + i)
		route.Link = router.DefaultLink
		if err := table.Create(route); err != nil {
			t.Fatalf("error adding route: %s", err)
		}
	}

	query = []router.QueryOption{
		router.QueryService("svcX"),
	}

	routes, err = table.Query(query...)
	if err != nil {
		t.Fatalf("error looking up routes: %s", err)
	}

	if len(routes) != 2 {
		t.Fatalf("incorrect number of routes returned. Expected: %d, found: %d", 1, len(routes))
	}
}

func TestFallback(t *testing.T) {

	r := &rtr{
		options: router.DefaultOptions(),
	}
	route := router.Route{
		Service: "go.micro.service.foo",
		Router:  r.options.Id,
		Link:    router.DefaultLink,
		Metric:  router.DefaultLocalMetric,
	}
	r.table = newTable(func(s string) ([]router.Route, error) {
		return []router.Route{route}, nil
	})
	r.start()

	rts, err := r.Lookup(router.QueryService("go.micro.service.foo"))
	if err != nil {
		t.Fatalf("error looking up service %s", err)
	}
	if len(rts) != 1 {
		t.Fatalf("incorrect number of routes returned %d", len(rts))
	}

	// deleting from the table but the next query should invoke the fallback that we passed during new table creation
	if err := r.table.Delete(route); err != nil {
		t.Fatalf("error deleting route %s", err)
	}

	rts, err = r.Lookup(router.QueryService("go.micro.service.foo"))
	if err != nil {
		t.Fatalf("error looking up service %s", err)
	}
	if len(rts) != 1 {
		t.Fatalf("incorrect number of routes returned %d", len(rts))
	}

}

func TestFallbackError(t *testing.T) {
	r := &rtr{
		options: router.DefaultOptions(),
	}
	r.table = newTable(func(s string) ([]router.Route, error) {
		return nil, fmt.Errorf("ERROR")
	})
	r.start()
	_, err := r.Lookup(router.QueryService("go.micro.service.foo"))
	if err == nil {
		t.Fatalf("expected error looking up service but none returned")
	}

}
