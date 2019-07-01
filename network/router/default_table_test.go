package router

import "testing"

func TestNewTable(t *testing.T) {
	table := NewTable()

	if table.Size() != 0 {
		t.Errorf("new table should be empty")
	}
}

func TestAdd(t *testing.T) {
	table := NewTable()

	if table.Size() != 0 {
		t.Errorf("new table should be empty")
	}

	route := Route{
		Destination: "dest.svc",
		Gateway:     "dest.gw",
		Router:      "dest.router",
		Network:     "dest.network",
		Metric:      10,
	}

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != 1 {
		t.Errorf("invalid number of routes. expected: 1, given: %d", table.Size())
	}

	// adds new route for the original destination
	route.Gateway = "dest.gw2"

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != 2 {
		t.Errorf("invalid number of routes. expected: 2, given: %d", table.Size())
	}

	// overrides an existing route: the size of the table does not change
	route.Metric = 100
	route.Policy = OverrideIfExists

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != 2 {
		t.Errorf("invalid number of routes. expected: 2, given: %d", table.Size())
	}

	// dont add new route if it already exists
	route.Policy = IgnoreIfExists

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != 2 {
		t.Errorf("invalid number of routes. expected: 2, given: %d", table.Size())
	}

	// adding the same route under this policy should error
	route.Policy = AddIfNotExists

	if err := table.Add(route); err != ErrDuplicateRoute {
		t.Errorf("error adding route. Expected error: %s, Given: %s", ErrDuplicateRoute, err)
	}
}

func TestDelete(t *testing.T) {
	table := NewTable()

	if table.Size() != 0 {
		t.Errorf("new table should be empty")
	}

	route := Route{
		Destination: "dest.svc",
		Gateway:     "dest.gw",
		Router:      "dest.router",
		Network:     "dest.network",
		Metric:      10,
	}

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != 1 {
		t.Errorf("invalid number of routes. expected: 1, given: %d", table.Size())
	}

	// should fail to delete non-existant route
	oldDest := route.Destination
	route.Destination = "randDest"

	if err := table.Delete(route); err != ErrRouteNotFound {
		t.Errorf("error deleting route. Expected error: %s, given: %s", ErrRouteNotFound, err)
	}

	if table.Size() != 1 {
		t.Errorf("invalid number of routes. expected: %d, given: %d", 1, table.Size())
	}

	// we should be able to delete the routes now
	route.Destination = oldDest

	if err := table.Delete(route); err != nil {
		t.Errorf("error deleting route: %s", err)
	}

	if table.Size() != 0 {
		t.Errorf("invalid number of routes. expected: %d, given: %d", 0, table.Size())
	}
}

func TestUpdate(t *testing.T) {
	table := NewTable()

	if table.Size() != 0 {
		t.Errorf("new table should be empty")
	}

	route := Route{
		Destination: "dest.svc",
		Gateway:     "dest.gw",
		Router:      "dest.router",
		Network:     "dest.network",
		Metric:      10,
	}

	if err := table.Add(route); err != nil {
		t.Errorf("error adding route: %s", err)
	}

	if table.Size() != 1 {
		t.Errorf("invalid number of routes. expected: 1, given: %d", table.Size())
	}

	route.Metric = 200

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}

	if table.Size() != 1 {
		t.Errorf("invalid number of routes. expected: 1, given: %d", table.Size())
	}

	// this should add a new route; we are hashing routes on <destination, gateway, network>
	route.Destination = "new.dest"

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}

	// NOTE: default policy is AddIfNotExists so the new route will be added here
	if table.Size() != 2 {
		t.Errorf("invalid number of routes. expected: 2, given: %d", table.Size())
	}

	// this should add a new route; we are hashing routes on <destination, gateway, network>
	route.Gateway = "new.gw"

	if err := table.Update(route); err != nil {
		t.Errorf("error updating route: %s", err)
	}

	// NOTE: default policy is AddIfNotExists so the new route will be added here
	if table.Size() != 3 {
		t.Errorf("invalid number of routes. expected: 3, given: %d", table.Size())
	}

	// this should NOTE add a new route; we are setting the policy to IgnoreIfExists
	route.Destination = "rand.dest"
	route.Policy = IgnoreIfExists

	if err := table.Update(route); err != ErrRouteNotFound {
		t.Errorf("error updating route. Expected error: %s, given: %s", ErrRouteNotFound, err)
	}

	if table.Size() != 3 {
		t.Errorf("invalid number of routes. expected: 3, given: %d", table.Size())
	}
}
