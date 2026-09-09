// Package api implements the 'micro api' command — a lightweight
// HTTP-to-RPC gateway that proxies JSON requests to go-micro services.
//
// Usage:
//
//	micro api                        # listen on :8080
//	micro api --address :3000        # custom port
//
// Requests:
//
//	POST /service/endpoint  →  RPC call to service.endpoint
//	GET  /health            →  {"status":"ok"}
//
// The request body is forwarded as-is (JSON). The Micro-Endpoint
// header can also be used to specify the endpoint.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/cmd"
	codecBytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "api",
		Usage: "Run a lightweight HTTP-to-RPC API gateway",
		Description: `Start an HTTP gateway that proxies JSON requests to go-micro services.

Requests are routed by URL path:
  POST /service/endpoint  →  calls service.endpoint via RPC
  GET  /                  →  lists available services and endpoints

Examples:
  # Start on default port
  micro api

  # Custom port
  micro api --address :3000

  # Call a service through the gateway
  curl -XPOST -d '{"name":"Alice"}' http://localhost:8080/greeter/Greeter.Hello

  # Or use the Micro-Endpoint header
  curl -XPOST -H 'Micro-Endpoint: Greeter.Hello' \
    -d '{"name":"Alice"}' http://localhost:8080/greeter`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Address to listen on",
				Value:   ":8080",
				EnvVars: []string{"MICRO_API_ADDRESS"},
			},
		},
		Action: run,
	})
}

func run(c *cli.Context) error {
	addr := c.String("address")

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Framework primitives under /micro/
	registerFrameworkRoutes(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := strings.TrimPrefix(r.URL.Path, "/")
		path = strings.TrimSuffix(path, "/")

		// Root: list services
		if path == "" {
			listServices(w)
			return
		}

		// Parse service/endpoint from path
		parts := strings.SplitN(path, "/", 2)
		serviceName := parts[0]
		endpoint := ""
		if len(parts) > 1 {
			endpoint = parts[1]
		}

		// Allow Micro-Endpoint header to override
		if h := r.Header.Get("Micro-Endpoint"); h != "" {
			endpoint = h
		}

		if endpoint == "" {
			describeService(w, serviceName)
			return
		}

		// Proxy RPC call
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read body: "+err.Error())
			return
		}
		if len(body) == 0 {
			body = []byte("{}")
		}

		req := client.DefaultClient.NewRequest(serviceName, endpoint, &codecBytes.Frame{Data: body})
		var rsp codecBytes.Frame

		if err := client.DefaultClient.Call(r.Context(), req, &rsp); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(rsp.Data)
	})

	fmt.Println()
	fmt.Println("  \033[1mmicro api\033[0m")
	fmt.Println()
	fmt.Printf("  Listening    \033[36m%s\033[0m\n", addr)
	fmt.Println()
	fmt.Println("  Routes:")
	fmt.Println("    \033[32mGET\033[0m  /                           List services")
	fmt.Println("    \033[32mGET\033[0m  /{service}                  Describe a service")
	fmt.Println("    \033[33mPOST\033[0m /{service}/{endpoint}       Call an endpoint")
	fmt.Println("    \033[32mGET\033[0m  /health                     Health check")
	fmt.Println()
	fmt.Println("  Framework:")
	fmt.Println("    \033[32mGET\033[0m  /micro/registry             List registered services")
	fmt.Println("    \033[32mGET\033[0m  /micro/registry/{name}      Describe a service")
	fmt.Println("    \033[32mGET\033[0m  /micro/store                List store keys")
	fmt.Println("    \033[32mGET\033[0m  /micro/store/{key}          Read a record")
	fmt.Println("    \033[33mPOST\033[0m /micro/store/{key}          Write a record")
	fmt.Println("    \033[33mPOST\033[0m /micro/broker/{topic}       Publish a message")
	fmt.Println()

	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	return server.Close()
}

func listServices(w http.ResponseWriter) {
	services, err := registry.ListServices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	type svcInfo struct {
		Name      string   `json:"name"`
		Endpoints []string `json:"endpoints,omitempty"`
	}

	var result []svcInfo
	for _, svc := range services {
		info := svcInfo{Name: svc.Name}
		full, err := registry.GetService(svc.Name)
		if err == nil && len(full) > 0 {
			for _, ep := range full[0].Endpoints {
				info.Endpoints = append(info.Endpoints, ep.Name)
			}
		}
		result = append(result, info)
	}

	json.NewEncoder(w).Encode(result)
}

func describeService(w http.ResponseWriter, name string) {
	services, err := registry.GetService(name)
	if err != nil || len(services) == 0 {
		writeError(w, http.StatusNotFound, "service not found: "+name)
		return
	}

	type epInfo struct {
		Name     string            `json:"name"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}

	svc := services[0]
	var endpoints []epInfo
	for _, ep := range svc.Endpoints {
		endpoints = append(endpoints, epInfo{
			Name:     ep.Name,
			Metadata: ep.Metadata,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"name":      svc.Name,
		"version":   svc.Version,
		"endpoints": endpoints,
		"nodes":     len(svc.Nodes),
	})
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// registerFrameworkRoutes adds /micro/* routes for registry, broker, and store.
func registerFrameworkRoutes(mux *http.ServeMux) {
	// Registry
	mux.HandleFunc("/micro/registry", func(w http.ResponseWriter, r *http.Request) {
		listServices(w)
	})
	mux.HandleFunc("/micro/registry/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/micro/registry/")
		if name == "" {
			listServices(w)
			return
		}
		describeService(w, name)
	})

	// Store
	mux.HandleFunc("/micro/store", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		keys, err := store.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		json.NewEncoder(w).Encode(keys)
	})
	mux.HandleFunc("/micro/store/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/micro/store/")
		if key == "" {
			w.Header().Set("Content-Type", "application/json")
			keys, _ := store.List()
			json.NewEncoder(w).Encode(keys)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			records, err := store.Read(key)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if len(records) == 0 {
				writeError(w, http.StatusNotFound, "key not found")
				return
			}
			w.Write(records[0].Value)
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			if err := store.Write(&store.Record{Key: key, Value: body}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "key": key})
		default:
			writeError(w, http.StatusMethodNotAllowed, "use GET or POST")
		}
	})

	// Broker
	mux.HandleFunc("/micro/broker/", func(w http.ResponseWriter, r *http.Request) {
		topic := strings.TrimPrefix(r.URL.Path, "/micro/broker/")
		if topic == "" {
			writeError(w, http.StatusBadRequest, "topic required: /micro/broker/{topic}")
			return
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "use POST to publish")
			return
		}
		body, _ := io.ReadAll(r.Body)
		b := broker.DefaultBroker
		if err := b.Connect(); err != nil {
			writeError(w, http.StatusInternalServerError, "broker connect: "+err.Error())
			return
		}
		if err := b.Publish(topic, &broker.Message{Body: body}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "topic": topic})
	})
}
