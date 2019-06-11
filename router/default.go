package router

import (
	"fmt"
	"strings"
	"sync"

	"github.com/micro/go-log"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/gossip"
	"github.com/olekukonko/tablewriter"
)

type router struct {
	opts Options
	goss registry.Registry
	exit chan struct{}
	wg   *sync.WaitGroup
}

func newRouter(opts ...Option) Router {
	// set default options
	options := Options{
		Table: NewTable(),
	}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// bind to gossip address to join gossip registry
	goss := gossip.NewRegistry(
		gossip.Address(options.GossipAddr),
	)

	return &router{
		opts: options,
		goss: goss,
		exit: make(chan struct{}),
		wg:   &sync.WaitGroup{},
	}
}

// Init initializes router with given options
func (r *router) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	return nil
}

// Options returns router options
func (r *router) Options() Options {
	return r.opts
}

// Table returns routing table
func (r *router) Table() Table {
	return r.opts.Table
}

// Address returns router's bind address
func (r *router) Address() string {
	return r.opts.Address
}

// Network returns router's micro network
func (r *router) Network() string {
	return r.opts.NetworkAddr
}

// Start starts the router
func (r *router) Start() error {
	// TODO:
	// - list all remote services and populate routing table
	// - list all local services and populate remote registry

	gWatcher, err := r.goss.Watch()
	if err != nil {
		return fmt.Errorf("failed to create router gossip registry watcher: %v", err)
	}

	tWatcher, err := r.opts.Table.Watch()
	if err != nil {
		return fmt.Errorf("failed to create routing table watcher: %v", err)
	}

	r.wg.Add(1)
	go r.watchGossip(gWatcher)

	r.wg.Add(1)
	go r.watchTable(tWatcher)

	return nil
}

// watch gossip registry
func (r *router) watchGossip(w registry.Watcher) error {
	defer r.wg.Done()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-r.exit
		// stop gossip registry watcher
		w.Stop()
	}()

	var watchErr error

	// watch for changes to services
	for {
		res, err := w.Next()
		if err == registry.ErrWatcherStopped {
			break
		}

		if err != nil {
			watchErr = err
			break
		}

		switch res.Action {
		case "create":
			if len(res.Service.Nodes) > 0 {
				log.Logf("Action: %s, Service: %v", res.Action, res.Service.Name)
			}
		case "delete":
			log.Logf("Action: %s, Service: %v", res.Action, res.Service.Name)
		}
	}

	return watchErr
}

// watch gossip registry
func (r *router) watchTable(w Watcher) error {
	defer r.wg.Done()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-r.exit
		// stop gossip registry watcher
		w.Stop()
	}()

	var watchErr error

	// watch for changes to services
	for {
		res, err := w.Next()
		if err == ErrWatcherStopped {
			break
		}

		if err != nil {
			watchErr = err
			break
		}

		switch res.Action {
		case "add":
			log.Logf("Action: %s, Route: %v", res.Action, res.Route)
		case "remove":
			log.Logf("Action: %s, Route: %v", res.Action, res.Route)
		}
	}

	return watchErr
}

// Stop stops the router
func (r *router) Stop() error {
	// notify all goroutines to finish
	close(r.exit)

	// wait for all goroutines to finish
	r.wg.Wait()

	return nil
}

// String prints debugging information about router
func (r *router) String() string {
	sb := &strings.Builder{}

	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"ID", "Address", "Gossip", "Network", "Table"})

	data := []string{
		r.opts.ID,
		r.opts.Address,
		r.opts.GossipAddr,
		r.opts.NetworkAddr,
		fmt.Sprintf("%d", r.opts.Table.Size()),
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
