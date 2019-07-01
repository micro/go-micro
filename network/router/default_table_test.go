package router

import "testing"

// creates routing table and test route
func testSetup() (Table, Route) {
	table := NewTable()

	route := Route{
		Destination: "dest.svc",
		Gateway:     "dest.gw",
		Router:      "dest.router",
		Network:     "dest.network",
		Metric:      10,
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

	// overrides an existing route
	// NOTE: the size of the table should not change
	route.Metric = 100
	route.Policy = OverrideIfExists

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// dont add new route if it already exists
	// NOTE: The size of the table should not change
	route.Policy = IgnoreIfExists

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// adding the same route under AddIfNotExists policy must error
	route.Policy = AddIfNotExists

	if err := table.Add(route); err != ErrDuplicateRoute {
		t.Errorf("error adding route. Expected error: %s, Given: %s", ErrDuplicateRoute, err)
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
	prevDest := route.Destination
	route.Destination = "randDest"

	if err := table.Delete(route); err != ErrRouteNotFound {
		t.Errorf("error deleting route. Expected error: %s, given: %s", ErrRouteNotFound, err)
	}

	// we should be able to delete the existing route
	route.Destination = prevDest

	if err := table.Delete(route); err != nil {
		t.Errorf("error deleting route: %s", err)
	}
	testTableSize -= 1

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
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
	// NOTE: this should NOT change the size of the table
	route.Metric = 200

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// NOTE: routing table routes on <destination, gateway, network>
	// this should add a new route
	route.Destination = "new.dest"

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}
	testTableSize += 1

	// NOTE: default policy is AddIfNotExists so the new route will be added here
	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// NOTE: we are hashing routes on <destination, gateway, network>
	// this should add a new route
	route.Gateway = "new.gw"

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}
	testTableSize += 1

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// this should NOT add a new route as we are setting the policy to IgnoreIfExists
	route.Destination = "rand.dest"
	route.Policy = IgnoreIfExists

	if err := table.Update(route); err != ErrRouteNotFound {
		t.Errorf("error updating route. Expected error: %s, given: %s", ErrRouteNotFound, err)
	}

	if table.Size() != 3 {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}
}

func TestList(t *testing.T) {
	table, route := testSetup()

	dest := []string{"one.svc", "two.svc", "three.svc"}

	for i := 0; i < len(dest); i++ {
		route.Destination = dest[i]
		if err := table.Add(route); err != nil {
			t.Errorf("error adding route: %s", err)
		}
	}

	routes, err := table.List()
	if err != nil {
		t.Errorf("error listing routes: %s", err)
	}

	if len(routes) != len(dest) {
		t.Errorf("incorrect number of routes listed. Expected: %d, Given: %d", len(dest), len(routes))
	}

	if len(routes) != table.Size() {
		t.Errorf("mismatch number of routes and table size. Routes: %d, Size: %d", len(routes), table.Size())
	}
}
