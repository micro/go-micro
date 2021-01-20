// Package http enables the http profiler
package http

import (
	"context"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/asim/go-micro/v3/debug/profile"
)

type httpProfile struct {
	sync.Mutex
	running bool
	server  *http.Server
}

var (
	DefaultAddress = ":6060"
)

// Start the profiler
func (h *httpProfile) Start() error {
	h.Lock()
	defer h.Unlock()

	if h.running {
		return nil
	}

	go func() {
		if err := h.server.ListenAndServe(); err != nil {
			h.Lock()
			h.running = false
			h.Unlock()
		}
	}()

	h.running = true

	return nil
}

// Stop the profiler
func (h *httpProfile) Stop() error {
	h.Lock()
	defer h.Unlock()

	if !h.running {
		return nil
	}

	h.running = false

	return h.server.Shutdown(context.TODO())
}

func (h *httpProfile) String() string {
	return "http"
}

func NewProfile(opts ...profile.Option) profile.Profile {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return &httpProfile{
		server: &http.Server{
			Addr:    DefaultAddress,
			Handler: mux,
		},
	}
}
