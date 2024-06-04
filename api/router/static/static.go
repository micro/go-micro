package static

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"go-micro.dev/v5/api/router"
	"go-micro.dev/v5/api/router/util"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/metadata"
	"go-micro.dev/v5/registry"
	rutil "go-micro.dev/v5/util/registry"
)

type endpoint struct {
	apiep    *router.Endpoint
	hostregs []*regexp.Regexp
	pathregs []util.Pattern
	pcreregs []*regexp.Regexp
}

// Router is the default router.
type Router struct {
	opts router.Options
	exit chan bool
	eps  map[string]*endpoint
	sync.RWMutex
}

func (r *Router) isStopd() bool {
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
		if r.isStopd() {
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

func (r *Router) Register(route *router.Route) error {
	myEndpoint := route.Endpoint

	if err := router.Validate(myEndpoint); err != nil {
		return err
	}

	var (
		pathregs []util.Pattern
		hostregs []*regexp.Regexp
		pcreregs []*regexp.Regexp
	)

	for _, h := range myEndpoint.Host {
		if h == "" || h == "*" {
			continue
		}
		hostreg, err := regexp.CompilePOSIX(h)
		if err != nil {
			return err
		}

		hostregs = append(hostregs, hostreg)
	}

	for _, p := range myEndpoint.Path {
		var pcreok bool

		// pcre only when we have start and end markers
		if p[0] == '^' && p[len(p)-1] == '$' {
			pcrereg, err := regexp.CompilePOSIX(p)
			if err == nil {
				pcreregs = append(pcreregs, pcrereg)
				pcreok = true
			}
		}

		rule, err := util.Parse(p)
		if err != nil && !pcreok {
			return err
		} else if err != nil && pcreok {
			continue
		}

		tpl := rule.Compile()

		pathreg, err := util.NewPattern(tpl.Version, tpl.OpCodes, tpl.Pool, "", util.PatternLogger(r.Options().Logger))
		if err != nil {
			return err
		}

		pathregs = append(pathregs, pathreg)
	}

	r.Lock()
	r.eps[myEndpoint.Name] = &endpoint{
		apiep:    myEndpoint,
		pcreregs: pcreregs,
		pathregs: pathregs,
		hostregs: hostregs,
	}
	r.Unlock()

	return nil
}

func (r *Router) Deregister(route *router.Route) error {
	ep := route.Endpoint
	if err := router.Validate(ep); err != nil {
		return err
	}

	r.Lock()
	delete(r.eps, ep.Name)
	r.Unlock()

	return nil
}

func (r *Router) Options() router.Options {
	return r.opts
}

func (r *Router) Stop() error {
	select {
	case <-r.exit:
		return nil
	default:
		close(r.exit)
	}

	return nil
}

func (r *Router) Endpoint(req *http.Request) (*router.Route, error) {
	myEndpoint, err := r.endpoint(req)
	if err != nil {
		return nil, err
	}

	epf := strings.Split(myEndpoint.apiep.Name, ".")

	services, err := r.opts.Registry.GetService(epf[0])
	if err != nil {
		return nil, err
	}

	// hack for stream endpoint
	if myEndpoint.apiep.Stream {
		svcs := rutil.Copy(services)
		for _, svc := range svcs {
			if len(svc.Endpoints) == 0 {
				e := &registry.Endpoint{}
				e.Name = strings.Join(epf[1:], ".")
				e.Metadata = make(map[string]string)
				e.Metadata["stream"] = "true"
				svc.Endpoints = append(svc.Endpoints, e)
			}

			for _, e := range svc.Endpoints {
				e.Name = strings.Join(epf[1:], ".")
				e.Metadata = make(map[string]string)
				e.Metadata["stream"] = "true"
			}
		}

		services = svcs
	}

	svc := &router.Route{
		Service: epf[0],
		Endpoint: &router.Endpoint{
			Name:    strings.Join(epf[1:], "."),
			Handler: "rpc",
			Host:    myEndpoint.apiep.Host,
			Method:  myEndpoint.apiep.Method,
			Path:    myEndpoint.apiep.Path,
			Stream:  myEndpoint.apiep.Stream,
		},
		Versions: services,
	}

	return svc, nil
}

func (r *Router) endpoint(req *http.Request) (*endpoint, error) {
	logger := r.Options().Logger

	if r.isStopd() {
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
	for _, myEndpoint := range r.eps {
		var mMatch, hMatch, pMatch bool

		// 1. try method
		for _, m := range myEndpoint.apiep.Method {
			if m == req.Method {
				mMatch = true
				break
			}
		}

		if !mMatch {
			continue
		}
		logger.Logf(log.DebugLevel, "api method match %s", req.Method)

		// 2. try host
		if len(myEndpoint.apiep.Host) == 0 {
			hMatch = true
		} else {
			for idx, h := range myEndpoint.apiep.Host {
				if h == "" || h == "*" {
					hMatch = true
					break
				} else if myEndpoint.hostregs[idx].MatchString(req.URL.Host) {
					hMatch = true
					break
				}
			}
		}

		if !hMatch {
			continue
		}

		logger.Logf(log.DebugLevel, "api host match %s", req.URL.Host)

		// 3. try google.api path
		for _, pathreg := range myEndpoint.pathregs {
			matches, err := pathreg.Match(path, "")
			if err != nil {
				logger.Logf(log.DebugLevel, "api gpath not match %s != %v", path, pathreg)
				continue
			}

			logger.Logf(log.DebugLevel, "api gpath match %s = %v", path, pathreg)

			pMatch = true
			ctx := req.Context()
			md, ok := metadata.FromContext(ctx)

			if !ok {
				md = make(metadata.Metadata)
			}

			for k, v := range matches {
				md[fmt.Sprintf("x-api-field-%s", k)] = v
			}

			*req = *req.Clone(metadata.NewContext(ctx, md))

			break
		}

		if !pMatch {
			// 4. try path via pcre path matching
			for _, pathreg := range myEndpoint.pcreregs {
				if !pathreg.MatchString(req.URL.Path) {
					logger.Logf(log.DebugLevel, "api pcre path not match %s != %v", req.URL.Path, pathreg)
					continue
				}

				pMatch = true

				break
			}
		}

		if !pMatch {
			continue
		}
		// TODO: Percentage traffic

		// we got here, so its a match
		return myEndpoint, nil
	}

	// no match
	return nil, fmt.Errorf("endpoint not found for %v", req.URL)
}

func (r *Router) Route(req *http.Request) (*router.Route, error) {
	if r.isStopd() {
		return nil, errors.New("router closed")
	}

	// try get an endpoint
	ep, err := r.Endpoint(req)
	if err != nil {
		return nil, err
	}

	return ep, nil
}

// NewRouter returns a new static router.
func NewRouter(opts ...router.Option) *Router {
	options := router.NewOptions(opts...)
	r := &Router{
		exit: make(chan bool),
		opts: options,
		eps:  make(map[string]*endpoint),
	}
	// go r.watch()
	return r
}
