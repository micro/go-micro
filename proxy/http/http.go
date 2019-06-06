// Package http provides a micro rpc to http proxy
package http

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/server"
)

// Router will proxy rpc requests as http POST requests. It is a server.Router
type Router struct {
	// The http backend to call
	Backend string

	// first request
	first bool
}

var (
	// The default backend
	DefaultBackend = "http://localhost:9090"
	// The default router
	DefaultRouter = &Router{}
)

func getMethod(hdr map[string]string) string {
	switch hdr["Micro-Method"] {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH":
		return hdr["Micro-Method"]
	default:
		return "POST"
	}
}

func getEndpoint(hdr map[string]string) string {
	ep := hdr["Micro-Endpoint"]
	if len(ep) > 0 && ep[0] == '/' {
		return ep
	}
	return ""
}

// ServeRequest honours the server.Router interface
func (p *Router) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	if p.Backend == "" {
		p.Backend = DefaultBackend
	}

	for {
		// get data
		body, err := req.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// get the header
		hdr := req.Header()

		// get method
		method := getMethod(hdr)

		// get endpoint
		endpoint := getEndpoint(hdr)

		// set the endpoint
		if len(endpoint) == 0 {
			endpoint = p.Backend
		} else {
			// add endpoint to backend
			u, err := url.Parse(p.Backend)
			if err != nil {
				return errors.InternalServerError(req.Service(), err.Error())
			}
			u.Path = path.Join(u.Path, endpoint)
			endpoint = u.String()
		}

		// send to backend
		hreq, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
		if err != nil {
			return errors.InternalServerError(req.Service(), err.Error())
		}

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

// NewSingleHostRouter returns a router which sends requests to a single http backend
//
// It is used by setting it in a new micro service to act as a proxy for a http backend.
//
// Usage:
//
// Create a new router to the http backend
//
// 	r := NewSingleHostRouter("http://localhost:10001")
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
		Backend: url,
	}
}

// NewService returns a new http proxy. It acts as a micro service proxy.
// Any request on the transport is routed to a fixed http backend.
//
// Usage:
//
// 	service := NewService(
//		micro.Name("greeter"),
//		// Sets the default http endpoint
//		http.WithBackend("http://localhost:10001"),
//	 )
//
func NewService(opts ...micro.Option) micro.Service {
	// prepend router to opts
	opts = append([]micro.Option{
		WithRouter(DefaultRouter),
	}, opts...)

	// create the new service
	return micro.NewService(opts...)
}
