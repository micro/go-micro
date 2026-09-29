// App serves a notes UI and the same notes capability over RPC.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"

	"go-micro.dev/v6/app"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/web"
)

//go:embed assets/*
var assets embed.FS

type AddRequest struct {
	Text string `json:"text" description:"Text of the note to save"`
}
type ListRequest struct{}
type NotesResponse struct {
	Notes []string `json:"notes"`
}
type Notes struct {
	mu    sync.Mutex
	notes []string
}

// Add saves a note and returns the current notes.
func (n *Notes) Add(ctx context.Context, req *AddRequest, rsp *NotesResponse) error {
	text := strings.TrimSpace(req.Text)
	if text == "" || len(text) > 4096 {
		return fmt.Errorf("note must contain 1 to 4096 bytes")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.notes = append(n.notes, text)
	rsp.Notes = append([]string{}, n.notes...)
	return nil
}

// List returns the saved notes.
func (n *Notes) List(ctx context.Context, req *ListRequest, rsp *NotesResponse) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	rsp.Notes = append([]string{}, n.notes...)
	return nil
}

func definition() app.Definition {
	return app.Definition{Name: "notes-app", Version: "1.0.0", Description: "Notes shared by a UI and agent tools", Entrypoint: "index.html", Services: []app.Dependency{{Name: "app-notes", Endpoints: []string{"Notes.List", "Notes.Add"}}}}
}

func newApplication(cl client.Client) (*app.App, http.Handler, error) {
	public, err := fs.Sub(assets, "assets")
	if err != nil {
		return nil, nil, err
	}
	a, err := app.New(definition(), public)
	if err != nil {
		return nil, nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/", a.Handler())
	// The host explicitly exposes two operations. The app package does not
	// introduce an unrestricted RPC proxy or derive permissions from its manifest.
	mux.HandleFunc("/api/notes", func(w http.ResponseWriter, r *http.Request) {
		var input any = &ListRequest{}
		endpoint := "Notes.List"
		switch r.Method {
		case http.MethodGet:
		case http.MethodPost:
			var req AddRequest
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil {
				http.Error(w, "invalid note", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(req.Text) == "" || len(req.Text) > 4096 {
				http.Error(w, "note must contain 1 to 4096 bytes", http.StatusBadRequest)
				return
			}
			input, endpoint = &req, "Notes.Add"
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var rsp NotesResponse
		if err := cl.Call(r.Context(), cl.NewRequest("app-notes", endpoint, input), &rsp); err != nil {
			http.Error(w, "notes service unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&rsp)
	})
	return a, mux, nil
}

func run(local bool) error {
	r := registry.DefaultRegistry
	cl := client.DefaultClient
	br := broker.DefaultBroker
	if local {
		r = registry.NewMemoryRegistry()
		br = broker.NewMemoryBroker()
		cl = client.NewClient(client.Registry(r), client.Selector(selector.NewSelector(selector.Registry(r))), client.Broker(br))
	}
	notes := service.New(service.Name("app-notes"), service.Address("127.0.0.1:0"), service.Registry(r), service.Client(cl), service.Broker(br), service.HandleSignal(false))
	if err := notes.Handle(new(Notes)); err != nil {
		return err
	}
	if err := notes.Start(); err != nil {
		return err
	}
	defer notes.Stop()
	a, handler, err := newApplication(cl)
	if err != nil {
		return err
	}
	if err := a.CheckDependencies(r); err != nil {
		return err
	}
	def := a.Definition()
	ui := web.NewService(web.Name(def.Name), web.Version(def.Version), web.Address("127.0.0.1:8080"), web.Registry(r), web.Handler(handler))
	if err := ui.Start(); err != nil {
		return err
	}
	defer ui.Stop()
	fmt.Println("Open http://127.0.0.1:8080 — notes are shared with the app-notes RPC service.")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	<-ctx.Done()
	return nil
}

func main() {
	local := flag.Bool("local", false, "Use in-process discovery when multicast is unavailable (external CLI discovery is disabled)")
	flag.Parse()
	if err := run(*local); err != nil {
		log.Fatal(err)
	}
}
