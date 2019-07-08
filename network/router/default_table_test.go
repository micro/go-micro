package router

import "testing"

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
	route.Metric = 100
	route.Policy = Override

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	// the size of the table should not change when Override policy is used
	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// dont add new route if it already exists
	route.Policy = Skip

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	// the size of the table should not change if Skip policy is used
	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// adding the same route under Insert policy must error
	route.Policy = Insert

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
	route.Metric = 200

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}

	// the size of the table should not change as we're only updating the metric of an existing route
	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// this should add a new route
	route.Destination = "new.dest"

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}
	testTableSize += 1

	// Default policy is Insert so the new route will be added here since the route does not exist
	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// this should add a new route
	route.Gateway = "new.gw"

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}
	testTableSize += 1

	if table.Size() != testTableSize {
		t.Errorf("invalid number of routes. expected: %d, given: %d", testTableSize, table.Size())
	}

	// this should NOT add a new route as we are setting the policy to Skip
	route.Destination = "rand.dest"
	route.Policy = Skip

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
