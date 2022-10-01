// Package web contains the web handler including websocket support
package web

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"go-micro.dev/v4/api/handler"
	"go-micro.dev/v4/api/router"
	"go-micro.dev/v4/selector"
)

const (
	// Handler is the name of the handler.
	Handler = "web"
)

type webHandler struct {
	opts handler.Options
}

func (wh *webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	service, err := wh.getService(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(service) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rp, err := url.Parse(service)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isWebSocket(r) {
		wh.serveWebSocket(rp.Host, w, r)
		return
	}

	httputil.NewSingleHostReverseProxy(rp).ServeHTTP(w, r)
}

// getService returns the service for this request from the selector.
func (wh *webHandler) getService(r *http.Request) (string, error) {
	var service *router.Route

	if wh.opts.Router != nil {
		// try get service from router
		s, err := wh.opts.Router.Route(r)
		if err != nil {
			return "", err
		}

		service = s
	} else {
		// we have no way of routing the request
		return "", errors.New("no route found")
	}

	// create a random selector
	next := selector.Random(service.Versions)

	// get the next node
	s, err := next()
	if err != nil {
		return "", nil
	}

	return fmt.Sprintf("http://%s", s.Address), nil
}

// serveWebSocket used to serve a web socket proxied connection.
func (wh *webHandler) serveWebSocket(host string, rsp http.ResponseWriter, r *http.Request) {
	req := new(http.Request)
	*req = *r

	if len(host) == 0 {
		http.Error(rsp, "invalid host", http.StatusInternalServerError)
		return
	}

	// set x-forward-for
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ips, ok := req.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(ips, ", ") + ", " + clientIP
		}

		req.Header.Set("X-Forwarded-For", clientIP)
	}

	// connect to the backend host
	conn, err := net.Dial("tcp", host)
	if err != nil {
		http.Error(rsp, err.Error(), http.StatusInternalServerError)
		return
	}

	// hijack the connection
	hj, ok := rsp.(http.Hijacker)
	if !ok {
		http.Error(rsp, "failed to connect", http.StatusInternalServerError)
		return
	}

	nc, _, err := hj.Hijack()
	if err != nil {
		return
	}

	defer nc.Close()
	defer conn.Close()

	if err = req.Write(conn); err != nil {
		return
	}

	errCh := make(chan error, 2)

	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errCh <- err
	}

	go cp(conn, nc)
	go cp(nc, conn)

	<-errCh
}

func isWebSocket(r *http.Request) bool {
	contains := func(key, val string) bool {
		vv := strings.Split(r.Header.Get(key), ",")
		for _, v := range vv {
			if val == strings.ToLower(strings.TrimSpace(v)) {
				return true
			}
		}

		return false
	}

	if contains("Connection", "upgrade") && contains("Upgrade", "websocket") {
		return true
	}

	return false
}

func (wh *webHandler) String() string {
	return "web"
}

// NewHandler returns a new web handler.
func NewHandler(opts ...handler.Option) handler.Handler {
	return &webHandler{
		opts: handler.NewOptions(opts...),
	}
}
