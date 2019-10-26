package router

import "github.com/hashicorp/go-memdb"

type memDBTable struct {
	schema *memdb.DBSchema
	db     *memdb.MemDB
}

func NewMemDBTable() *memDBTable {
	return &memDBTable{}
}

// Create new route in the routing table
func (t *memDBTable) Create(Route) error {
	return nil
}

// Delete existing route from the routing table
func (t *memDBTable) Delete(Route) error {
	return nil
}

// Update route in the routing table
func (t *memDBTable) Update(Route) error {
	return nil
}

// List all routes in the table
func (t *memDBTable) List() ([]Route, error) {
	return nil, nil
}

// Query routes in the routing table
func (t *memDBTable) Query(...QueryOption) ([]Route, error) {
	return nil, nil
}

// Watch wates routes in the routing table
func (t *memDBTable) Watch(...WatchOption) (Watcher, error) {
	return nil, nil
}
