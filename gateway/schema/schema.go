// Package schema provides a unified service schema resolver shared by the HTTP
// API gateway and the MCP gateway. It owns registry watching and endpoint
// parsing so both gateways query one cached catalog instead of duplicating
// discovery and reflection logic.
package schema

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v6/registry"
)

// Field describes one request or response field of an endpoint.
type Field struct {
	Name string
	Type string
}

// Endpoint is the resolved schema for one service endpoint, keyed by its
// dotted name "Service.Endpoint" (e.g. "helloworld.Helloworld.Call").
type Endpoint struct {
	// Service is the registered service name (e.g. "helloworld").
	Service string
	// Name is the dotted endpoint name (e.g. "helloworld.Helloworld.Call").
	Name string
	// Method is the registry endpoint name (e.g. "Helloworld.Call").
	Method string
	// Description is the endpoint description from metadata, or a default.
	Description string
	// Request lists the endpoint's request fields.
	Request []Field
	// Response lists the endpoint's response fields.
	Response []Field
	// Scopes is the comma-separated scope requirement from endpoint metadata.
	Scopes []string
	// Example is an example input from endpoint metadata, if any.
	Example string
	// Metadata is the raw endpoint metadata.
	Metadata map[string]string
}

// Resolver watches a registry and caches the endpoint catalog so REST proxying
// and MCP tool cataloging share one source of service metadata.
type Resolver struct {
	reg       registry.Registry
	logger    *log.Logger
	mu        sync.RWMutex
	services  map[string]*registry.Service // service name -> latest snapshot
	endpoints map[string]*Endpoint         // dotted name -> endpoint schema
	changes   chan struct{}
	startOnce sync.Once
}

// New returns a resolver bound to reg.
func New(reg registry.Registry) *Resolver {
	return &Resolver{
		reg:       reg,
		logger:    log.Default(),
		services:  map[string]*registry.Service{},
		endpoints: map[string]*Endpoint{},
		changes:   make(chan struct{}, 1),
	}
}

// WithLogger sets the logger used for registry watch errors.
func (r *Resolver) WithLogger(l *log.Logger) *Resolver {
	if l != nil {
		r.logger = l
	}
	return r
}

// refreshInterval is a fallback poll so the catalog self-heals if a watch
// event is dropped (registries deliver events asynchronously and may miss
// bursts that arrive while a refresh is in progress).
const refreshInterval = 15 * time.Second

// Start refreshes the catalog once, then watches the registry and re-refreshes
// on every change until ctx is done. The watcher is registered before Start
// returns so no events are missed. Safe to call once.
func (r *Resolver) Start(ctx context.Context) {
	r.startOnce.Do(func() {
		if _, err := r.Refresh(); err != nil {
			r.logger.Printf("[schema] initial registry refresh failed: %v", err)
		}
		watcher, err := r.reg.Watch()
		if err != nil {
			r.logger.Printf("[schema] registry watch failed: %v", err)
			return
		}
		go r.watch(ctx, watcher)
	})
}

func (r *Resolver) watch(ctx context.Context, watcher registry.Watcher) {
	defer watcher.Stop()

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.Refresh(); err != nil {
				r.logger.Printf("[schema] registry refresh failed: %v", err)
			}
			continue
		default:
		}
		if _, err := watcher.Next(); err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		changed, err := r.Refresh()
		if err != nil {
			r.logger.Printf("[schema] registry refresh failed: %v", err)
			continue
		}
		if !changed {
			continue
		}
		select {
		case r.changes <- struct{}{}:
		default:
		}
	}
}

// Refresh re-reads the registry and rebuilds the catalog. It returns true if
// the catalog actually changed. It does not signal Changes(); only the internal
// watch loop does, so callers that refresh from their own handlers (e.g. the
// MCP gateway rediscovering tools) do not loop.
func (r *Resolver) Refresh() (bool, error) {
	services, err := r.reg.ListServices()
	if err != nil {
		return false, err
	}

	snapshot := make(map[string]*registry.Service, len(services))
	endpoints := make(map[string]*Endpoint)
	for _, svc := range services {
		full, err := r.reg.GetService(svc.Name)
		if err != nil || len(full) == 0 {
			continue
		}
		snapshot[svc.Name] = full[0]
		for _, ep := range full[0].Endpoints {
			e := resolveEndpoint(svc.Name, ep)
			endpoints[e.Name] = e
		}
	}

	r.mu.Lock()
	changed := !equalEndpoints(r.endpoints, endpoints)
	r.services = snapshot
	r.endpoints = endpoints
	r.mu.Unlock()
	return changed, nil
}

// equalEndpoints reports whether two endpoint maps have the same keys and
// identical endpoint metadata (service, method, scopes).
func equalEndpoints(a, b map[string]*Endpoint) bool {
	if len(a) != len(b) {
		return false
	}
	for name, ea := range a {
		eb, ok := b[name]
		if !ok {
			return false
		}
		if ea.Service != eb.Service || ea.Method != eb.Method || !equalScopes(ea.Scopes, eb.Scopes) {
			return false
		}
	}
	return true
}

func equalScopes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func resolveEndpoint(service string, ep *registry.Endpoint) *Endpoint {
	e := &Endpoint{
		Service:  service,
		Name:     service + "." + ep.Name,
		Method:   ep.Name,
		Metadata: ep.Metadata,
	}
	e.Description = fmt.Sprintf("Call %s on %s service", ep.Name, service)
	if ep.Metadata != nil {
		if d, ok := ep.Metadata["description"]; ok && d != "" {
			e.Description = d
		}
		if scopes, ok := ep.Metadata["scopes"]; ok && scopes != "" {
			for _, scope := range strings.Split(scopes, ",") {
				if scope = strings.TrimSpace(scope); scope != "" {
					e.Scopes = append(e.Scopes, scope)
				}
			}
		}
		if example, ok := ep.Metadata["example"]; ok {
			e.Example = example
		}
	}
	e.Request = fieldsOf(ep.Request)
	e.Response = fieldsOf(ep.Response)
	return e
}

func fieldsOf(v *registry.Value) []Field {
	if v == nil {
		return nil
	}
	out := make([]Field, 0, len(v.Values))
	for _, f := range v.Values {
		out = append(out, Field{Name: f.Name, Type: f.Type})
	}
	return out
}

// Endpoints returns the catalog, sorted by dotted name.
func (r *Resolver) Endpoints() []*Endpoint {
	r.mu.RLock()
	out := make([]*Endpoint, 0, len(r.endpoints))
	for _, e := range r.endpoints {
		out = append(out, e)
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Endpoint returns the schema for a dotted endpoint name.
func (r *Resolver) Endpoint(name string) (*Endpoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.endpoints[name]
	return e, ok
}

// EndpointsFor returns the endpoints of a service, sorted by dotted name.
func (r *Resolver) EndpointsFor(service string) []*Endpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Endpoint
	for _, e := range r.endpoints {
		if e.Service == service {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Service returns the latest registry snapshot for a service, or nil.
func (r *Resolver) Service(name string) *registry.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[name]
}

// Services returns the registered service names, sorted.
func (r *Resolver) Services() []string {
	r.mu.RLock()
	out := make([]string, 0, len(r.services))
	for name := range r.services {
		out = append(out, name)
	}
	r.mu.RUnlock()
	sort.Strings(out)
	return out
}

// HasService reports whether a service is currently registered.
func (r *Resolver) HasService(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.services[name]
	return ok
}

// Changes returns a channel that signals each catalog refresh from the watch
// loop. The initial refresh from Start is not signaled.
func (r *Resolver) Changes() <-chan struct{} {
	return r.changes
}

// JSONType maps a Go type to a JSON schema type. Shared so REST and MCP
// gateways emit identical endpoint schemas.
func JSONType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	}
	if strings.HasPrefix(goType, "[]") {
		return "array"
	}
	return "object"
}
