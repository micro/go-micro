// Package http provides a micro to http proxy
package http

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/errors"
	"github.com/asim/go-micro/v3/server"
)

// Router will proxy rpc requests as http POST requests. It is a server.Router
type Router struct {
	// Converts RPC Foo.Bar to /foo/bar
	Resolver *Resolver
	// The http backend to call
	Backend string

	// first request
	first bool
	// rpc ep / http ep mapping
	eps map[string]string
}

// Resolver resolves rpc to http. It explicity maps Foo.Bar to /foo/bar
type Resolver struct{}

var (
	// The default backend
	DefaultBackend = "http://localhost:9090"
	// The default router
	DefaultRouter = &Router{}
)

// Foo.Bar becomes /foo/bar
func (r *Resolver) Resolve(ep string) string {
	// replace . with /
	ep = strings.Replace(ep, ".", "/", -1)
	// lowercase the whole thing
	ep = strings.ToLower(ep)
	// prefix with "/"
	return filepath.Join("/", ep)
}

// set the nil things
func (p *Router) setup() {
	if p.Resolver == nil {
		p.Resolver = new(Resolver)
	}
	if p.Backend == "" {
		p.Backend = DefaultBackend
	}
	if p.eps == nil {
		p.eps = map[string]string{}
	}
}

// Endpoint returns the http endpoint for an rpc endpoint.
// Endpoint("Foo.Bar") returns http://localhost:9090/foo/bar
func (p *Router) Endpoint(rpcEp string) (string, error) {
	p.setup()

	// get http endpoint
	ep, ok := p.eps[rpcEp]
	if !ok {
		// get default
		ep = p.Resolver.Resolve(rpcEp)
	}

	// already full qualified URL
	if strings.HasPrefix(ep, "http://") || strings.HasPrefix(ep, "https://") {
		return ep, nil
	}

	// parse into url

	// full path to call
	u, err := url.Parse(p.Backend)
	if err != nil {
		return "", err
	}

	// set path
	u.Path = filepath.Join(u.Path, ep)

	// set scheme
	if len(u.Scheme) == 0 {
		u.Scheme = "http"
	}

	// set host
	if len(u.Host) == 0 {
		u.Host = "localhost"
	}

	// create ep
	return u.String(), nil
}

// RegisterEndpoint registers a http endpoint against an RPC endpoint.
// It converts relative paths into backend:endpoint. Anything prefixed
// with http:// or https:// will be left as is.
//	RegisterEndpoint("Foo.Bar", "/foo/bar")
//	RegisterEndpoint("Greeter.Hello", "/helloworld")
//	RegisterEndpoint("Greeter.Hello", "http://localhost:8080/")
func (p *Router) RegisterEndpoint(rpcEp, httpEp string) error {
	p.setup()

	// create ep
	p.eps[rpcEp] = httpEp
	return nil
}

func (p *Router) ProcessMessage(ctx context.Context, msg server.Message) error {
	return nil
}

// ServeRequest honours the server.Router interface
func (p *Router) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	// rudimentary post based streaming
	for {
		// get data
		body, err := req.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var rpcEp string

		// get rpc endpoint
		if p.first {
			p.first = false
			rpcEp = req.Endpoint()
		} else {
			hdr := req.Header()
			rpcEp = hdr["X-Micro-Endpoint"]
		}

		// get http endpoint
		ep, err := p.Endpoint(rpcEp)
		if err != nil {
			return errors.NotFound(req.Service(), err.Error())
		}

		// no stream support currently
		// TODO: lookup host
		hreq, err := http.NewRequest("POST", ep, bytes.NewReader(body))
		if err != nil {
			return errors.InternalServerError(req.Service(), err.Error())
		}

		// get the header
		hdr := req.Header()

		// set the headers
		for k, v := range hdr {
			hreq.Header.Set(k, v)
		}

		// make the call
		hrsp, err := http.DefaultClient.Do(hreq)
		if err != nil {
			return errors.InternalServerError(req.Service(), err.Error())
		}

		// read body
		b, err := ioutil.ReadAll(hrsp.Body)
		hrsp.Body.Close()
		if err != nil {
			return errors.InternalServerError(req.Service(), err.Error())
		}

		// set response headers
		hdr = map[string]string{}
		for k, _ := range hrsp.Header {
			hdr[k] = hrsp.Header.Get(k)
		}
		// write the header
		rsp.WriteHeader(hdr)
		// write the body
		err = rsp.Write(b)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.InternalServerError(req.Service(), err.Error())
		}
	}

	return nil
}

// NewSingleHostRouter returns a router which sends requests a single http backend
//
// It is used by setting it in a new micro service to act as a proxy for a http backend.
//
// Usage:
//
// Create a new router to the http backend
//
// 	r := NewSingleHostRouter("http://localhost:10001")
//
//	// Add additional routes
//	r.RegisterEndpoint("Hello.World", "/helloworld")
//
// 	// Create your new service
// 	service := micro.NewService(
// 		micro.Name("greeter"),
//		// Set the router
//		http.WithRouter(r),
// 	)
//
// 	// Run the service
// 	service.Run()
func NewSingleHostRouter(url string) *Router {
	return &Router{
		Resolver: new(Resolver),
		Backend:  url,
		eps:      map[string]string{},
	}
}

// NewService returns a new http proxy. It acts as a micro service and proxies to a http backend.
// Routes are dynamically set e.g Foo.Bar routes to /foo/bar. The default backend is http://localhost:9090.
// Optionally specify the backend endpoint url or the router. Also choose to register specific endpoints.
//
// Usage:
//
// 	service := NewService(
//		micro.Name("greeter"),
//		// Sets the default http endpoint
//		http.WithBackend("http://localhost:10001"),
//	 )
//
// Set fixed backend endpoints
//
//	// register an endpoint
//	http.RegisterEndpoint("Hello.World", "/helloworld")
//
// 	service := NewService(
//		micro.Name("greeter"),
//		// Set the http endpoint
//		http.WithBackend("http://localhost:10001"),
//	 )
func NewService(opts ...micro.Option) micro.Service {
	// prepend router to opts
	opts = append([]micro.Option{
		WithRouter(DefaultRouter),
	}, opts...)

	// create the new service
	return micro.NewService(opts...)
}

// RegisterEndpoint registers a http endpoint against an RPC endpoint
//	RegisterEndpoint("Foo.Bar", "/foo/bar")
//	RegisterEndpoint("Greeter.Hello", "/helloworld")
//	RegisterEndpoint("Greeter.Hello", "http://localhost:8080/")
func RegisterEndpoint(rpcEp string, httpEp string) error {
	return DefaultRouter.RegisterEndpoint(rpcEp, httpEp)
}
