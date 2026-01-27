// Package gateway provides an HTTP gateway for micro run
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5/client"
	"go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/health"
	"go-micro.dev/v5/registry"
)

// Gateway provides HTTP access to micro services
type Gateway struct {
	addr     string
	server   *http.Server
	services []ServiceInfo
	mu       sync.RWMutex
}

// ServiceInfo holds information about a running service
type ServiceInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port,omitempty"`
}

// New creates a new gateway
func New(addr string) *Gateway {
	return &Gateway{
		addr: addr,
	}
}

// SetServices updates the list of known services
func (g *Gateway) SetServices(services []ServiceInfo) {
	g.mu.Lock()
	g.services = services
	g.mu.Unlock()
}

// Start starts the gateway HTTP server
func (g *Gateway) Start() error {
	mux := http.NewServeMux()

	// Health endpoint - aggregates all service health
	mux.HandleFunc("/health", g.healthHandler)
	mux.HandleFunc("/health/live", g.liveHandler)
	mux.HandleFunc("/health/ready", g.readyHandler)

	// API endpoint - HTTP to RPC proxy
	mux.HandleFunc("/api/", g.apiHandler)

	// Services list
	mux.HandleFunc("/services", g.servicesHandler)

	// Home page
	mux.HandleFunc("/", g.homeHandler)

	g.server = &http.Server{
		Addr:    g.addr,
		Handler: mux,
	}

	go func() {
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Gateway error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the gateway
func (g *Gateway) Stop() {
	if g.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		g.server.Shutdown(ctx)
	}
}

// Addr returns the gateway address
func (g *Gateway) Addr() string {
	return g.addr
}

func (g *Gateway) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	g.mu.RLock()
	services := g.services
	g.mu.RUnlock()

	// Get services from registry
	regServices, _ := registry.ListServices()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Micro</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; }
        .container { max-width: 800px; margin: 0 auto; padding: 40px 20px; }
        h1 { font-size: 2em; margin-bottom: 10px; }
        .subtitle { color: #666; margin-bottom: 30px; }
        .card { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .card h2 { font-size: 1.2em; margin-bottom: 15px; color: #333; }
        .service { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid #eee; }
        .service:last-child { border-bottom: none; }
        .service-name { font-weight: 500; }
        .service-addr { color: #666; font-family: monospace; font-size: 0.9em; }
        .endpoints { margin-top: 10px; }
        .endpoint { display: block; padding: 5px 10px; margin: 5px 0; background: #f0f0f0; border-radius: 4px; font-family: monospace; font-size: 0.85em; text-decoration: none; color: #333; }
        .endpoint:hover { background: #e0e0e0; }
        .try-it { background: #f9f9f9; padding: 15px; border-radius: 6px; margin-top: 20px; }
        .try-it h3 { font-size: 1em; margin-bottom: 10px; }
        code { background: #333; color: #0f0; padding: 10px 15px; display: block; border-radius: 4px; font-size: 0.85em; overflow-x: auto; }
        .links { margin-top: 20px; }
        .links a { color: #0066cc; margin-right: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Micro</h1>
        <p class="subtitle">Services are running</p>
        
        <div class="card">
            <h2>Services (%d)</h2>
`, len(regServices))

	if len(regServices) > 0 {
		for _, svc := range regServices {
			fmt.Fprintf(w, `            <div class="service">
                <span class="service-name">%s</span>
            </div>
`, svc.Name)

			// Get endpoints for this service
			if details, err := registry.GetService(svc.Name); err == nil && len(details) > 0 {
				if len(details[0].Endpoints) > 0 {
					fmt.Fprintf(w, `            <div class="endpoints">`)
					for _, ep := range details[0].Endpoints {
						fmt.Fprintf(w, `                <a class="endpoint" href="/api/%s/%s">POST /api/%s/%s</a>\n`,
							svc.Name, ep.Name, svc.Name, ep.Name)
					}
					fmt.Fprintf(w, `            </div>`)
				}
			}
		}
	} else if len(services) > 0 {
		for _, svc := range services {
			fmt.Fprintf(w, `            <div class="service">
                <span class="service-name">%s</span>
                <span class="service-addr">%s</span>
            </div>
`, svc.Name, svc.Address)
		}
	} else {
		fmt.Fprintf(w, `            <p style="color: #666; padding: 10px 0;">No services registered yet...</p>`)
	}

	fmt.Fprintf(w, `        </div>
        
        <div class="card">
            <h2>Quick Links</h2>
            <div class="links">
                <a href="/health">Health Check</a>
                <a href="/services">Services JSON</a>
            </div>
        </div>

        <div class="try-it">
            <h3>Try it</h3>
            <code>curl -X POST http://localhost%s/api/{service}/{Endpoint} -d '{}'</code>
        </div>
    </div>
</body>
</html>`, g.addr)
}

func (g *Gateway) servicesHandler(w http.ResponseWriter, r *http.Request) {
	services, err := registry.ListServices()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var result []map[string]interface{}
	for _, svc := range services {
		details, _ := registry.GetService(svc.Name)
		var endpoints []string
		if len(details) > 0 {
			for _, ep := range details[0].Endpoints {
				endpoints = append(endpoints, ep.Name)
			}
		}
		result = append(result, map[string]interface{}{
			"name":      svc.Name,
			"endpoints": endpoints,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (g *Gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := health.Run(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if resp.Status == health.StatusUp {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *Gateway) liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"up"}`))
}

func (g *Gateway) readyHandler(w http.ResponseWriter, r *http.Request) {
	g.healthHandler(w, r)
}

func (g *Gateway) apiHandler(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/{service}/{endpoint}
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 2 {
		http.Error(w, `{"error": "usage: /api/{service}/{endpoint}"}`, http.StatusBadRequest)
		return
	}

	service := parts[0]
	endpoint := parts[1]

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	// Create RPC request
	req := client.NewRequest(service, endpoint, &bytes.Frame{Data: body})

	var rsp bytes.Frame
	if err := client.Call(r.Context(), req, &rsp); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(rsp.Data)
}
