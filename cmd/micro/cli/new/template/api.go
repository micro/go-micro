package template

var (
	ApiProtoSRV = `syntax = "proto3";

package {{dehyphen .Alias}};

option go_package = "./proto;{{dehyphen .Alias}}";

service {{title .Alias}} {
	rpc Health(HealthRequest) returns (HealthResponse) {}
	rpc Endpoint(EndpointRequest) returns (EndpointResponse) {}
}

message HealthRequest {}

message HealthResponse {
	string status = 1;
	int64 uptime = 2;
}

message EndpointRequest {
	string method = 1;
	string path = 2;
	string body = 3;
	map<string, string> headers = 4;
}

message EndpointResponse {
	int32 status_code = 1;
	string body = 2;
	map<string, string> headers = 3;
}
`

	ApiHandlerSRV = `package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "go-micro.dev/v6/logger"

	pb "{{.Dir}}/proto"
)

type {{title .Alias}} struct {
	started time.Time
	routes  map[string]http.HandlerFunc
}

func New() *{{title .Alias}} {
	h := &{{title .Alias}}{
		started: time.Now(),
		routes:  make(map[string]http.HandlerFunc),
	}
	h.registerRoutes()
	return h
}

func (h *{{title .Alias}}) registerRoutes() {
	h.routes["GET /hello"] = func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Hello %s", name),
		})
	}
}

// Health returns the service health status and uptime.
//
// @example {}
func (h *{{title .Alias}}) Health(ctx context.Context, req *pb.HealthRequest, rsp *pb.HealthResponse) error {
	rsp.Status = "ok"
	rsp.Uptime = int64(time.Since(h.started).Seconds())
	return nil
}

// Endpoint handles proxied HTTP requests. The method and path fields
// select the route; body and headers are forwarded.
//
// @example {"method": "GET", "path": "/hello", "body": "", "headers": {}}
func (h *{{title .Alias}}) Endpoint(ctx context.Context, req *pb.EndpointRequest, rsp *pb.EndpointResponse) error {
	key := fmt.Sprintf("%s %s", req.Method, req.Path)
	handler, ok := h.routes[key]
	if !ok {
		log.Infof("Route not found: %s", key)
		rsp.StatusCode = 404
		rsp.Body = ` + "`" + `{"error":"not found"}` + "`" + `
		return nil
	}

	rec := &responseRecorder{headers: make(map[string]string), statusCode: 200}
	fakeReq, _ := http.NewRequestWithContext(ctx, req.Method, req.Path, nil)
	handler(rec, fakeReq)

	rsp.StatusCode = int32(rec.statusCode)
	rsp.Body = rec.body
	rsp.Headers = rec.headers
	return nil
}

type responseRecorder struct {
	headers    map[string]string
	body       string
	statusCode int
}

func (r *responseRecorder) Header() http.Header         { return http.Header{} }
func (r *responseRecorder) WriteHeader(statusCode int)   { r.statusCode = statusCode }
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = string(b)
	return len(b), nil
}
`
)
