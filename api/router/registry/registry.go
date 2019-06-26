// Package registry provides a dynamic api service router
package registry

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/api"
	"github.com/micro/go-micro/api/router"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/cache"
)

// router is the default router
type registryRouter struct {
	exit chan bool
	opts router.Options

	// registry cache
	rc cache.Cache

	sync.RWMutex
	eps map[string]*api.Service
}

func setNamespace(ns, name string) string {
	ns = strings.TrimSpace(ns)
	name = strings.TrimSpace(name)

	// no namespace
	if len(ns) == 0 {
		return name
	}

	switch {
	// has - suffix
	case strings.HasSuffix(ns, "-"):
		return strings.Replace(ns+name, ".", "-", -1)
	// has . suffix
	case strings.HasSuffix(ns, "."):
		return ns + name
	}

	// default join .
	return strings.Join([]string{ns, name}, ".")
}

func (r *registryRouter) isClosed() bool {
	select {
	case <-r.exit:
		return true
	default:
		return false
	}
}

// refresh list of api services
func (r *registryRouter) refresh() {
	var attempts int

	for {
		services, err := r.opts.Registry.ListServices()
		if err != nil {
			attempts++
			log.Println("Error listing endpoints", err)
			time.Sleep(time.Duration(attempts) * time.Second)
			continue
		}

		attempts = 0

		// for each service, get service and store endpoints
		for _, s := range services {
			// only get services for this namespace
			if !strings.HasPrefix(s.Name, r.opts.Namespace) {
				continue
			}
			service, err := r.rc.GetService(s.Name)
			if err != nil {
				continue
			}
			r.store(service)
		}

		// refresh list in 10 minutes... cruft
		select {
		case <-time.After(time.Minute * 10):
		case <-r.exit:
			return
		}
	}
}

// process watch event
func (r *registryRouter) process(res *registry.Result) {
	// skip these things
	if res == nil || res.Service == nil || !strings.HasPrefix(res.Service.Name, r.opts.Namespace) {
		return
	}

	// get entry from cache
	service, err := r.rc.GetService(res.Service.Name)
	if err != nil {
		return
	}

	// update our local endpoints
	r.store(service)
}

// store local endpoint cache
func (r *registryRouter) store(services []*registry.Service) {
	// endpoints
	eps := map[string]*api.Service{}

	// services
	names := map[string]bool{}

	// create a new endpoint mapping
	for _, service := range services {
		// set names we need later
		names[service.Name] = true

		// map per endpoint
		for _, endpoint := range service.Endpoints {
			// create a key service:endpoint_name
			key := fmt.Sprintf("%s:%s", service.Name, endpoint.Name)
			// decode endpoint
			end := api.Decode(endpoint.Metadata)

			// if we got nothing skip
			if err := api.Validate(end); err != nil {
				continue
			}

			// try get endpoint
			ep, ok := eps[key]
			if !ok {
				ep = &api.Service{Name: service.Name}
			}

			// overwrite the endpoint
			ep.Endpoint = end
			// append services
			ep.Services = append(ep.Services, service)
			// store it
			eps[key] = ep
		}
	}

	r.Lock()
	defer r.Unlock()

	// delete any existing eps for services we know
	for key, service := range r.eps {
		// skip what we don't care about
		if !names[service.Name] {
			continue
		}

		// ok we know this thing
		// delete delete delete
		delete(r.eps, key)
	}

	// now set the eps we have
	for name, endpoint := range eps {
		r.eps[name] = endpoint
	}
}

// watch for endpoint changes
func (r *registryRouter) watch() {
	var attempts int

	for {
		if r.isClosed() {
			return
		}

		// watch for changes
		w, err := r.opts.Registry.Watch()
		if err != nil {
			attempts++
			log.Println("Error watching endpoints", err)
			time.Sleep(time.Duration(attempts) * time.Second)
			continue
		}

		ch := make(chan bool)

		go func() {
			select {
			case <-ch:
				w.Stop()
			case <-r.exit:
				w.Stop()
			}
		}()

		// reset if we get here
		attempts = 0

		for {
			// process next event
			res, err := w.Next()
			if err != nil {
				log.Println("Error getting next endpoint", err)
				close(ch)
				break
			}
			r.process(res)
		}
	}
}

func (r *registryRouter) Options() router.Options {
	return r.opts
}

func (r *registryRouter) Close() error {
	select {
	case <-r.exit:
		return nil
	default:
		close(r.exit)
		r.rc.Stop()
	}
	return nil
}

func (r *registryRouter) Endpoint(req *http.Request) (*api.Service, error) {
	if r.isClosed() {
		return nil, errors.New("router closed")
	}

	r.RLock()
	defer r.RUnlock()

	// use the first match
	// TODO: weighted matching
	for _, e := range r.eps {
		ep := e.Endpoint

		// match
		var pathMatch, hostMatch, methodMatch bool

		// 1. try method GET, POST, PUT, etc
		// 2. try host example.com, foobar.com, etc
		// 3. try path /foo/bar, /bar/baz, etc

		// 1. try match method
		for _, m := range ep.Method {
			if req.Method == m {
				methodMatch = true
				break
			}
		}

		// no match on method pass
		if len(ep.Method) > 0 && !methodMatch {
			continue
		}

		// 2. try match host
		for _, h := range ep.Host {
			if req.Host == h {
				hostMatch = true
				break
			}
		}

		// no match on host pass
		if len(ep.Host) > 0 && !hostMatch {
			continue
		}

		// 3. try match paths
		for _, p := range ep.Path {
			re, err := regexp.CompilePOSIX(p)
			if err == nil && re.MatchString(req.URL.Path) {
				pathMatch = true
				break
			}
		}

		// no match pass
		if len(ep.Path) > 0 && !pathMatch {
			continue
		}

		// TODO: Percentage traffic

		// we got here, so its a match
		return e, nil
	}

	// no match
	return nil, errors.New("not found")
}

func (r *registryRouter) Route(req *http.Request) (*api.Service, error) {
	if r.isClosed() {
		return nil, errors.New("router closed")
	}

	// try get an endpoint
	ep, err := r.Endpoint(req)
	if err == nil {
		return ep, nil
	}

	// error not nil
	// ignore that shit
	// TODO: don't ignore that shit

	// get the service name
	rp, err := r.opts.Resolver.Resolve(req)
	if err != nil {
		return nil, err
	}

	// service name
	name := setNamespace(r.opts.Namespace, rp.Name)

	// get service
	services, err := r.rc.GetService(name)
	if err != nil {
		return nil, err
	}

	// only use endpoint matching when the meta handler is set aka api.Default
	switch r.opts.Handler {
	// rpc handlers
	case "meta", "api", "rpc":
		handler := r.opts.Handler

		// set default handler to api
		if r.opts.Handler == "meta" {
			handler = "rpc"
		}

		// construct api service
		return &api.Service{
			Name: name,
			Endpoint: &api.Endpoint{
				Name:    rp.Method,
				Handler: handler,
			},
			Services: services,
		}, nil
	// http handler
	case "http", "proxy", "web":
		// construct api service
		return &api.Service{
			Name: name,
			Endpoint: &api.Endpoint{
				Name:    req.URL.String(),
				Handler: r.opts.Handler,
				Host:    []string{req.Host},
				Method:  []string{req.Method},
				Path:    []string{req.URL.Path},
			},
			Services: services,
		}, nil
	}

	return nil, errors.New("unknown handler")
}

func newRouter(opts ...router.Option) *registryRouter {
	options := router.NewOptions(opts...)
	r := &registryRouter{
		exit: make(chan bool),
		opts: options,
		rc:   cache.New(options.Registry),
		eps:  make(map[string]*api.Service),
	}
	go r.watch()
	go r.refresh()
	return r
}

// NewRouter returns the default router
func NewRouter(opts ...router.Option) router.Router {
	return newRouter(opts...)
}
