package router

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
	"github.com/micro/go-micro/util/log"
)

type memDBTable struct {
	// schema defines routing table schmea
	schema *memdb.DBSchema
	// db stores routes in MemDB
	db *memdb.MemDB

	sync.RWMutex
	// watchers stores table watchers
	watchers map[string]*tableWatcher
}

func NewMemDBTable() *memDBTable {
	schema := tableSchema()
	if err := schema.Validate(); err != nil {
		panic(err)
	}
	// Create a new MemDB instance
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	return &memDBTable{
		schema:   schema,
		db:       db,
		watchers: make(map[string]*tableWatcher),
	}
}

// sendEvent sends events to all subscribed watchers
func (t *memDBTable) sendEvent(e *Event) {
	t.RLock()
	defer t.RUnlock()

	for _, w := range t.watchers {
		select {
		case w.resChan <- e:
		case <-w.done:
		}
	}
}

func (t *memDBTable) queryExactRoute(r Route) error {
	txn := t.db.Txn(false)
	defer txn.Abort()

	route, err := txn.First("route", "id", r.Service, r.Address, r.Gateway, r.Network, r.Router, r.Link)
	if err != nil {
		return err
	}

	if route != nil {
		return ErrDuplicateRoute
	}

	return nil
}

// Create creates new route in the routing table
func (t *memDBTable) Create(r Route) error {
	if err := t.queryExactRoute(r); err != nil {
		return err
	}

	// Create a write transaction
	txn := t.db.Txn(true)
	if err := txn.Insert("route", r); err != nil {
		txn.Abort()
		return err
	}
	// Commit the transaction
	txn.Commit()

	log.Debugf("Router MemDB emitting %s for route: %s", Create, r.Address)
	go t.sendEvent(&Event{Type: Create, Timestamp: time.Now(), Route: r})

	return nil
}

// Delete deletes the route from the routing table
func (t *memDBTable) Delete(r Route) error {
	// Create a write transaction
	txn := t.db.Txn(true)
	if err := txn.Delete("route", r); err != nil {
		txn.Abort()
		// we wra; memdb error into ours
		if err == memdb.ErrNotFound {
			return ErrRouteNotFound
		}
		return err
	}
	// Commit the transaction
	txn.Commit()

	log.Debugf("Router MemDB emitting %s for route: %s", Delete, r.Address)
	go t.sendEvent(&Event{Type: Delete, Timestamp: time.Now(), Route: r})

	return nil
}

// Update updates routing table with the new route
func (t *memDBTable) Update(r Route) error {
	err := t.queryExactRoute(r)
	if err != nil && err != ErrDuplicateRoute {
		return err
	}

	// Create a write transaction
	txn := t.db.Txn(true)
	if err := txn.Insert("route", r); err != nil {
		txn.Abort()
		return err
	}
	// Commit the transaction
	txn.Commit()

	// Only emit the event if the route never existed
	if err != ErrDuplicateRoute {
		log.Debugf("Router MemDB emitting %s for route: %s", Update, r.Address)
		go t.sendEvent(&Event{Type: Update, Timestamp: time.Now(), Route: r})
	}

	return nil
}

// List returns a list of all routes in the table
func (t *memDBTable) List() ([]Route, error) {
	txn := t.db.Txn(false)
	defer txn.Abort()

	// List all the routes
	r, err := txn.Get("route", "id")
	if err != nil {
		return nil, err
	}

	var routes []Route
	for obj := r.Next(); obj != nil; obj = r.Next() {
		route := obj.(Route)
		routes = append(routes, route)
	}

	return routes, nil
}

func matchedRoutes(res memdb.ResultIterator, opts QueryOptions) []Route {
	match := func(a, b string) bool {
		if a == "*" || a == b {
			return true
		}
		return false
	}

	var routes []Route
	for obj := res.Next(); obj != nil; obj = res.Next() {
		route := obj.(Route)
		values := []struct {
			a string
			b string
		}{
			{opts.Gateway, route.Gateway},
			{opts.Network, route.Network},
			{opts.Router, route.Router},
			{opts.Address, route.Address},
		}

		found := true
		for _, v := range values {
			// attempt to match each value
			if !match(v.a, v.b) {
				found = false
				break
			}
		}
		if found {
			routes = append(routes, route)
		}
	}

	return routes
}

// Query routes in the routing table
func (t *memDBTable) Query(q ...QueryOption) ([]Route, error) {
	// create new query options
	opts := NewQuery(q...)

	txn := t.db.Txn(false)
	defer txn.Abort()

	var res memdb.ResultIterator
	var err error

	switch opts.Service {
	case "*":
		res, err = txn.Get("route", "id")
	default:
		res, err = txn.Get("route", "service", opts.Service)
	}

	if err != nil {
		return nil, err
	}

	routes := matchedRoutes(res, opts)

	if len(routes) == 0 {
		return nil, ErrRouteNotFound
	}

	return routes, nil
}

// Watch wates routes in the routing table
func (t *memDBTable) Watch(opts ...WatchOption) (Watcher, error) {
	// by default watch everything
	wopts := WatchOptions{
		Service: "*",
	}

	for _, o := range opts {
		o(&wopts)
	}

	w := &tableWatcher{
		id:      uuid.New().String(),
		opts:    wopts,
		resChan: make(chan *Event, 10),
		done:    make(chan struct{}),
	}

	// when the watcher is stopped delete it
	go func() {
		<-w.done
		t.Lock()
		delete(t.watchers, w.id)
		t.Unlock()
	}()

	// save the watcher
	t.Lock()
	t.watchers[w.id] = w
	t.Unlock()

	return w, nil
}

func tableSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"route": &memdb.TableSchema{
				Name: "route",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{
									Field:     "Service",
									Lowercase: true,
								},
								&memdb.StringFieldIndex{
									Field:     "Address",
									Lowercase: true,
								},
								&memdb.StringFieldIndex{
									Field:     "Gateway",
									Lowercase: true,
								},
								&memdb.StringFieldIndex{
									Field:     "Network",
									Lowercase: true,
								},
								&memdb.StringFieldIndex{
									Field:     "Router",
									Lowercase: true,
								},
								&memdb.StringFieldIndex{
									Field:     "Link",
									Lowercase: true,
								},
							},
						},
					},
					"service": &memdb.IndexSchema{
						Name:         "service",
						AllowMissing: false,
						Unique:       false,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Service",
							Lowercase: true,
						},
					},
					"address": &memdb.IndexSchema{
						Name:         "address",
						AllowMissing: false,
						Unique:       false,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Address",
							Lowercase: true,
						},
					},
					"gateway": &memdb.IndexSchema{
						Name:         "gateway",
						AllowMissing: true,
						Unique:       false,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Gateway",
							Lowercase: true,
						},
					},
					"network": &memdb.IndexSchema{
						Name:         "network",
						AllowMissing: false,
						Unique:       false,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Network",
							Lowercase: true,
						},
					},
					"router": &memdb.IndexSchema{
						Name:         "router",
						AllowMissing: false,
						Unique:       false,
						Indexer: &memdb.StringFieldIndex{
							Field:     "Router",
							Lowercase: true,
						},
					},
				},
			},
		},
	}

}
