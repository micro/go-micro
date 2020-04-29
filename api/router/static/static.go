package static

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/micro/go-micro/v2/api"
	"github.com/micro/go-micro/v2/api/router"
	"github.com/micro/go-micro/v2/api/router/util"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/registry"
	rutil "github.com/micro/go-micro/v2/util/registry"
)

type endpoint struct {
	apiep    *api.Endpoint
	hostregs []*regexp.Regexp
	pathregs []util.Pattern
	pcreregs []*regexp.Regexp
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

	var pathregs []util.Pattern
	var hostregs []*regexp.Regexp
	var pcreregs []*regexp.Regexp

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
		pathreg, err := util.NewPattern(tpl.Version, tpl.OpCodes, tpl.Pool, "")
		if err != nil {
			return err
		}
		pathregs = append(pathregs, pathreg)
	}

	r.Lock()
	r.eps[ep.Name] = &endpoint{
		apiep:    ep,
		pcreregs: pcreregs,
		pathregs: pathregs,
		hostregs: hostregs,
	}
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

	svc := &api.Service{
		Name: epf[0],
		Endpoint: &api.Endpoint{
			Name:    strings.Join(epf[1:], "."),
			Handler: "rpc",
			Host:    ep.apiep.Host,
			Method:  ep.apiep.Method,
			Path:    ep.apiep.Path,
			Body:    ep.apiep.Body,
			Stream:  ep.apiep.Stream,
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
		for _, m := range ep.apiep.Method {
			if m == req.Method {
				mMatch = true
				break
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
			for idx, h := range ep.apiep.Host {
				if h == "" || h == "*" {
					hMatch = true
					break
				} else {
					if ep.hostregs[idx].MatchString(req.URL.Host) {
						hMatch = true
						break
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

		// 3. try google.api path
		for _, pathreg := range ep.pathregs {
			matches, err := pathreg.Match(path, "")
			if err != nil {
				if logger.V(logger.DebugLevel, logger.DefaultLogger) {
					logger.Debugf("api gpath not match %s != %v", path, pathreg)
				}
				continue
			}
			if logger.V(logger.DebugLevel, logger.DefaultLogger) {
				logger.Debugf("api gpath match %s = %v", path, pathreg)
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
			md["x-api-body"] = ep.apiep.Body
			*req = *req.Clone(metadata.NewContext(ctx, md))
			break
		}

		if !pMatch {
			// 4. try path via pcre path matching
			for _, pathreg := range ep.pcreregs {
				if !pathreg.MatchString(req.URL.Path) {
					if logger.V(logger.DebugLevel, logger.DefaultLogger) {
						logger.Debugf("api pcre path not match %s != %v", req.URL.Path, pathreg)
					}
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
		return ep, nil
	}

	// no match
	return nil, fmt.Errorf("endpoint not found for %v", req.URL)
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
