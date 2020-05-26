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

	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/proxy"
	"github.com/micro/go-micro/v2/server"
)

// Proxy will proxy rpc requests as http POST requests. It is a server.Proxy
type Proxy struct {
	options proxy.Options

	// The http backend to call
	Endpoint string

	// first request
	first bool
}

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

func getTopic(hdr map[string]string) string {
	ep := hdr["Micro-Topic"]
	if len(ep) > 0 && ep[0] == '/' {
		return ep
	}
	return "/" + hdr["Micro-Topic"]
}

// ProcessMessage handles incoming asynchronous messages
func (p *Proxy) ProcessMessage(ctx context.Context, msg server.Message) error {
	if p.Endpoint == "" {
		p.Endpoint = proxy.DefaultEndpoint
	}

	// get the header
	hdr := msg.Header()

	// get topic
	// use /topic as endpoint
	endpoint := getTopic(hdr)

	// set the endpoint
	if len(endpoint) == 0 {
		endpoint = p.Endpoint
	} else {
		// add endpoint to backend
		u, err := url.Parse(p.Endpoint)
		if err != nil {
			return errors.InternalServerError(msg.Topic(), err.Error())
		}
		u.Path = path.Join(u.Path, endpoint)
		endpoint = u.String()
	}

	// send to backend
	hreq, err := http.NewRequest("POST", endpoint, bytes.NewReader(msg.Body()))
	if err != nil {
		return errors.InternalServerError(msg.Topic(), err.Error())
	}

	// set the headers
	for k, v := range hdr {
		hreq.Header.Set(k, v)
	}

	// make the call
	hrsp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return errors.InternalServerError(msg.Topic(), err.Error())
	}

	// read body
	b, err := ioutil.ReadAll(hrsp.Body)
	hrsp.Body.Close()
	if err != nil {
		return errors.InternalServerError(msg.Topic(), err.Error())
	}

	if hrsp.StatusCode != 200 {
		return errors.New(msg.Topic(), string(b), int32(hrsp.StatusCode))
	}

	return nil
}

// ServeRequest honours the server.Router interface
func (p *Proxy) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	if p.Endpoint == "" {
		p.Endpoint = proxy.DefaultEndpoint
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
			endpoint = p.Endpoint
		} else {
			// add endpoint to backend
			u, err := url.Parse(p.Endpoint)
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
		for k := range hrsp.Header {
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
}

func (p *Proxy) String() string {
	return "http"
}

// NewSingleHostProxy returns a router which sends requests to a single http backend
func NewSingleHostProxy(url string) proxy.Proxy {
	return &Proxy{
		Endpoint: url,
	}
}

// NewProxy returns a new proxy which will route using a http client
func NewProxy(opts ...proxy.Option) proxy.Proxy {
	var options proxy.Options
	for _, o := range opts {
		o(&options)
	}

	p := new(Proxy)
	p.Endpoint = options.Endpoint
	p.options = options

	return p
}
