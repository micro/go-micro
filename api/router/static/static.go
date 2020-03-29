package static

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/httprule"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/micro/go-micro/v2/api"
	"github.com/micro/go-micro/v2/api/router"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/metadata"
)

type endpoint struct {
	apiep    *api.Endpoint
	hostregs []*regexp.Regexp
	pathregs []runtime.Pattern
}

// router is the default router
type staticRouter struct {
	exit chan bool
	opts router.Options
	sync.RWMutex
	eps map[string]*endpoint
}

func (r *staticRouter) isClosed() bool {
	select {
	case <-r.exit:
		return true
	default:
		return false
	}
}

/*
// watch for endpoint changes
func (r *staticRouter) watch() {
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
*/

func (r *staticRouter) Register(ep *api.Endpoint) error {
	if err := api.Validate(ep); err != nil {
		return err
	}

	var pathregs []runtime.Pattern
	var hostregs []*regexp.Regexp

	for _, h := range ep.Host {
		if h == "" || h == "*" {
			continue
		}
		hostreg, err := regexp.CompilePOSIX(h)
		if err != nil {
			return err
		}
		hostregs = append(hostregs, hostreg)
	}

	for _, p := range ep.Path {
		rule, err := httprule.Parse(p)
		if err != nil {
			return err
		}
		tpl := rule.Compile()
		pathreg, err := runtime.NewPattern(tpl.Version, tpl.OpCodes, tpl.Pool, "")
		if err != nil {
			return err
		}
		pathregs = append(pathregs, pathreg)
	}

	r.Lock()
	r.eps[ep.Name] = &endpoint{apiep: ep, pathregs: pathregs, hostregs: hostregs}
	r.Unlock()
	return nil
}

func (r *staticRouter) Deregister(ep *api.Endpoint) error {
	if err := api.Validate(ep); err != nil {
		return err
	}
	r.Lock()
	delete(r.eps, ep.Name)
	r.Unlock()
	return nil
}

func (r *staticRouter) Options() router.Options {
	return r.opts
}

func (r *staticRouter) Close() error {
	select {
	case <-r.exit:
		return nil
	default:
		close(r.exit)
	}
	return nil
}

func (r *staticRouter) Endpoint(req *http.Request) (*api.Service, error) {
	ep, err := r.endpoint(req)
	if err != nil {
		return nil, err
	}

	epf := strings.Split(ep.apiep.Name, ".")
	services, err := r.opts.Registry.GetService(epf[0])
	if err != nil {
		return nil, err
	}

	// hack for stream endpoint
	if ep.apiep.Stream {
		for _, svc := range services {
			for _, e := range svc.Endpoints {
				e.Name = strings.Join(epf[1:], ".")
				e.Metadata = make(map[string]string)
				e.Metadata["stream"] = "true"
			}
		}
	}

	svc := &api.Service{
		Name: epf[0],
		Endpoint: &api.Endpoint{
			Name:    strings.Join(epf[1:], "."),
			Handler: "rpc",
			Host:    ep.apiep.Host,
			Method:  ep.apiep.Method,
			Path:    ep.apiep.Path,
		},
		Services: services,
	}

	return svc, nil
}

func (r *staticRouter) endpoint(req *http.Request) (*endpoint, error) {
	if r.isClosed() {
		return nil, errors.New("router closed")
	}

	r.RLock()
	defer r.RUnlock()

	var idx int
	if len(req.URL.Path) > 0 && req.URL.Path != "/" {
		idx = 1
	}
	path := strings.Split(req.URL.Path[idx:], "/")
	// use the first match
	// TODO: weighted matching

	for _, ep := range r.eps {
		var mMatch, hMatch, pMatch bool

		// 1. try method
	methodLoop:
		for _, m := range ep.apiep.Method {
			if m == req.Method {
				mMatch = true
				break methodLoop
			}
		}
		if !mMatch {
			continue
		}
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("api method match %s", req.Method)
		}

		// 2. try host
		if len(ep.apiep.Host) == 0 {
			hMatch = true
		} else {
		hostLoop:
			for idx, h := range ep.apiep.Host {
				if h == "" || h == "*" {
					hMatch = true
					break hostLoop
				} else {
					if ep.hostregs[idx].MatchString(req.URL.Host) {
						hMatch = true
						break hostLoop
					}
				}
			}
		}
		if !hMatch {
			continue
		}
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("api host match %s", req.URL.Host)
		}

		// 3. try path
	pathLoop:
		for _, pathreg := range ep.pathregs {
			matches, err := pathreg.Match(path, "")
			if err != nil {
				// TODO: log error
				continue
			}
			pMatch = true
			ctx := req.Context()
			md, ok := metadata.FromContext(ctx)
			if !ok {
				md = make(metadata.Metadata)
			}
			for k, v := range matches {
				md[fmt.Sprintf("x-api-field-%s", k)] = v
			}
			*req = *req.WithContext(context.WithValue(ctx, metadata.MetadataKey{}, md))
			//req = req.WithContext(metadata.NewContext(ctx, md))
			break pathLoop
		}
		if !pMatch {
			continue
		}
		// TODO: Percentage traffic

		// we got here, so its a match
		return ep, nil
	}
	// no match
	return nil, fmt.Errorf("endpoint not found for %v", req)
}

func (r *staticRouter) Route(req *http.Request) (*api.Service, error) {
	if r.isClosed() {
		return nil, errors.New("router closed")
	}

	// try get an endpoint
	ep, err := r.Endpoint(req)
	if err != nil {
		return nil, err
	}

	return ep, nil
}

func NewRouter(opts ...router.Option) *staticRouter {
	options := router.NewOptions(opts...)
	r := &staticRouter{
		exit: make(chan bool),
		opts: options,
		eps:  make(map[string]*endpoint),
	}
	//go r.watch()
	//go r.refresh()
	return r
}
