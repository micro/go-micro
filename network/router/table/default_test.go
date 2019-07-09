package table

import "testing"

func testSetup() (Table, Route) {
	table := NewTable()

	route := Route{
		Service: "dest.svc",
		Gateway: "dest.gw",
		Network: "dest.network",
		Link:    "det.link",
		Metric:  10,
	}

	return table, route
}

func TestAdd(t *testing.T) {
	table, route := testSetup()
	testTableSize := table.Size()

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}
	testTableSize += 1

	// adds new route for the original destination
	route.Gateway = "dest.gw2"

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}
	testTableSize += 1

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. Expected: %d, found: %d", testTableSize, table.Size())
	}

	// adding the same route under Insert policy must error
	if err := table.Add(route); err != ErrDuplicateRoute {
		t.Errorf("error adding route. Expected error: %s, found: %s", ErrDuplicateRoute, err)
	}
}

func TestDelete(t *testing.T) {
	table, route := testSetup()
	testTableSize := table.Size()

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}
	testTableSize += 1

	// should fail to delete non-existant route
	prevSvc := route.Service
	route.Service = "randDest"

	if err := table.Delete(route); err != ErrRouteNotFound {
		t.Errorf("error deleting route. Expected: %s, found: %s", ErrRouteNotFound, err)
	}

	// we should be able to delete the existing route
	route.Service = prevSvc

	if err := table.Delete(route); err != nil {
		t.Errorf("error deleting route: %s", err)
	}
	testTableSize -= 1

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. Expected: %d, found: %d", testTableSize, table.Size())
	}
}

func TestUpdate(t *testing.T) {
	table, route := testSetup()
	testTableSize := table.Size()

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}
	testTableSize += 1

	// change the metric of the original route
	route.Metric = 200

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}

	// the size of the table should not change as we're only updating the metric of an existing route
	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. Expected: %d, found: %d", testTableSize, table.Size())
	}

	// this should error as the destination does not exist
	route.Service = "rand.dest"

	if err := table.Update(route); err != ErrRouteNotFound {
		t.Errorf("error updating route. Expected error: %s, found: %s", ErrRouteNotFound, err)
	}

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. Expected: %d, found: %d", testTableSize, table.Size())
	}
}

func TestList(t *testing.T) {
	table, route := testSetup()

	svc := []string{"one.svc", "two.svc", "three.svc"}

	for i := 0; i < len(svc); i++ {
		route.Service = svc[i]
		if err := table.Add(route); err != nil {
			t.Errorf("error adding route: %s", err)
		}
	}

	routes, err := table.List()
	if err != nil {
		t.Errorf("error listing routes: %s", err)
	}

	if len(routes) != len(svc) {
		t.Errorf("incorrect number of routes listed. Expected: %d, found: %d", len(svc), len(routes))
	}

	if len(routes) != table.Size() {
		t.Errorf("mismatch number of routes and table size. Expected: %d, found: %d", len(routes), table.Size())
	}
}

func TestLookup(t *testing.T) {
	table, route := testSetup()

	svc := []string{"svc1", "svc2", "svc3"}
	net := []string{"net1", "net2", "net1"}
	gw := []string{"gw1", "gw2", "gw3"}

	for i := 0; i < len(svc); i++ {
		route.Service = svc[i]
		route.Network = net[i]
		route.Gateway = gw[i]
		if err := table.Add(route); err != nil {
			t.Errorf("error adding route: %s", err)
		}
	}

	// return all routes
	query := NewQuery()

	routes, err := table.Lookup(query)
	if err != nil {
		t.Errorf("error looking up routes: %s", err)
	}

	if len(routes) != table.Size() {
		t.Errorf("incorrect number of routes returned. Expected: %d, found: %d", table.Size(), len(routes))
	}

	// query particular net
	query = NewQuery(QueryNetwork("net1"))

	routes, err = table.Lookup(query)
	if err != nil {
		t.Errorf("error looking up routes: %s", err)
	}

	if len(routes) != 2 {
		t.Errorf("incorrect number of routes returned. Expected: %d, found: %d", 2, len(routes))
	}

	// query particular gateway
	gateway := "gw1"
	query = NewQuery(QueryGateway(gateway))

	routes, err = table.Lookup(query)
	if err != nil {
		t.Errorf("error looking up routes: %s", err)
	}

	if len(routes) != 1 {
		t.Errorf("incorrect number of routes returned. Expected: %d, found: %d", 1, len(routes))
	}

	if routes[0].Gateway != gateway {
		t.Errorf("incorrect route returned. Expected gateway: %s, found: %s", gateway, routes[0].Gateway)
	}

	// query particular route
	network := "net1"
	query = NewQuery(
		QueryGateway(gateway),
		QueryNetwork(network),
	)

	routes, err = table.Lookup(query)
	if err != nil {
		t.Errorf("error looking up routes: %s", err)
	}

	if len(routes) != 1 {
		t.Errorf("incorrect number of routes returned. Expected: %d, found: %d", 1, len(routes))
	}

	if routes[0].Gateway != gateway {
		t.Errorf("incorrect route returned. Expected gateway: %s, found: %s", gateway, routes[0].Gateway)
	}

	if routes[0].Network != network {
		t.Errorf("incorrect network returned. Expected network: %s, found: %s", network, routes[0].Network)
	}

	// bullshit route query
	query = NewQuery(QueryService("foobar"))

	routes, err = table.Lookup(query)
	if err != nil {
		t.Errorf("error looking up routes: %s", err)
	}

	if len(routes) != 0 {
		t.Errorf("incorrect number of routes returned. Expected: %d, found: %d", 0, len(routes))
	}
}
