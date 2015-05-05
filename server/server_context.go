package server

import (
	"net/http"
	"sync"

	log "github.com/golang/glog"
	"github.com/myodc/go-micro/client"
)

var ctxs = struct {
	sync.Mutex
	m map[*http.Request]*serverContext
}{
	m: make(map[*http.Request]*serverContext),
}

// A server context interface
type Context interface {
	Request() Request                                           // the request made to the server
	Headers() Headers                                           // the response headers
	NewRequest(string, string, interface{}) client.Request      // a new scoped client request
	NewProtoRequest(string, string, interface{}) client.Request // a new scoped client request
	NewJsonRequest(string, string, interface{}) client.Request  // a new scoped client request
}

// context represents the context of an in-flight HTTP request.
// It implements the appengine.Context and http.ResponseWriter interfaces.
type serverContext struct {
	req       *serverRequest
	outCode   int
	outHeader http.Header
	outBody   []byte
}

// Copied from $GOROOT/src/pkg/net/http/transfer.go. Some response status
// codes do not permit a response body (nor response entity headers such as
// Content-Length, Content-Type, etc).
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

func getServerContext(req *http.Request) *serverContext {
	ctxs.Lock()
	c := ctxs.m[req]
	ctxs.Unlock()

	if c == nil {
		// Someone passed in an http.Request that is not in-flight.
		panic("NewContext passed an unknown http.Request")
	}
	return c
}

func (c *serverContext) NewRequest(service, method string, request interface{}) client.Request {
	req := client.NewRequest(service, method, request)
	// TODO: set headers and scope
	req.Headers().Set("X-User-Session", c.Request().Session("X-User-Session"))
	return req
}

func (c *serverContext) NewProtoRequest(service, method string, request interface{}) client.Request {
	req := client.NewProtoRequest(service, method, request)
	// TODO: set headers and scope
	req.Headers().Set("X-User-Session", c.Request().Session("X-User-Session"))
	return req
}

func (c *serverContext) NewJsonRequest(service, method string, request interface{}) client.Request {
	req := client.NewJsonRequest(service, method, request)
	// TODO: set headers and scope
	req.Headers().Set("X-User-Session", c.Request().Session("X-User-Session"))
	return req
}

// The response headers
func (c *serverContext) Headers() Headers {
	return c.outHeader
}

// The response headers
func (c *serverContext) Header() http.Header {
	return c.outHeader
}

// The request made to the server
func (c *serverContext) Request() Request {
	return c.req
}

func (c *serverContext) Write(b []byte) (int, error) {
	if c.outCode == 0 {
		c.WriteHeader(http.StatusOK)
	}
	if len(b) > 0 && !bodyAllowedForStatus(c.outCode) {
		return 0, http.ErrBodyNotAllowed
	}
	c.outBody = append(c.outBody, b...)
	return len(b), nil
}

func (c *serverContext) WriteHeader(code int) {
	if c.outCode != 0 {
		log.Error("WriteHeader called multiple times on request.")
		return
	}
	c.outCode = code
}

func GetContext(r *http.Request) *serverContext {
	return getServerContext(r)
}
