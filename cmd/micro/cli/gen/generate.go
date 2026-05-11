// Package generate provides code generation commands for micro
package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
)

var handlerTemplate = `package handler

import (
	"context"
	"fmt"

	log "go-micro.dev/v5/logger"
)

{{range .Methods}}
// {{.RequestType}} is the input for {{$.Name}}.{{.Name}}
type {{.RequestType}} struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name,omitempty\"`" + `
}

// {{.ResponseType}} is the output for {{$.Name}}.{{.Name}}
type {{.ResponseType}} struct {
	ID      string ` + "`json:\"id\"`" + `
	Message string ` + "`json:\"message\"`" + `
}
{{end}}

type {{.Name}} struct{}

func New{{.Name}}() *{{.Name}} {
	return &{{.Name}}{}
}

{{range .Methods}}
// {{.Name}} handles {{$.Name}}.{{.Name}} requests.
//
// @example {"id": "1", "name": "test"}
func (h *{{$.Name}}) {{.Name}}(ctx context.Context, req *{{.RequestType}}, rsp *{{.ResponseType}}) error {
	log.Infof("Received {{$.Name}}.{{.Name}} request: id=%s", req.ID)

	if req.ID == "" {
		return fmt.Errorf("id is required")
	}

	rsp.ID = req.ID
	rsp.Message = fmt.Sprintf("{{.Name}} processed: %s", req.Name)
	return nil
}
{{end}}
`

var endpointTemplate = `package handler

import (
	"encoding/json"
	"net/http"

	log "go-micro.dev/v5/logger"
)

// {{.Name}}Request is the request for {{.Name}}
type {{.Name}}Request struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name,omitempty\"`" + `
}

// {{.Name}}Response is the response for {{.Name}}
type {{.Name}}Response struct {
	ID      string ` + "`json:\"id\"`" + `
	Message string ` + "`json:\"message\"`" + `
	OK      bool   ` + "`json:\"ok\"`" + `
}

// {{.Name}} handles HTTP {{.Method}} requests to /{{.Path}}
func {{.Name}}(w http.ResponseWriter, r *http.Request) {
	log.Infof("Received {{.Name}} %s request", r.Method)

	var req {{.Name}}Request
	if r.Method == http.MethodGet {
		req.ID = r.URL.Query().Get("id")
		req.Name = r.URL.Query().Get("name")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, ` + "`" + `{"error":"invalid request body"}` + "`" + `, http.StatusBadRequest)
			return
		}
	}

	if req.ID == "" {
		http.Error(w, ` + "`" + `{"error":"id is required"}` + "`" + `, http.StatusBadRequest)
		return
	}

	rsp := {{.Name}}Response{
		ID:      req.ID,
		Message: "processed",
		OK:      true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rsp)
}
`

var modelTemplate = `package model

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// {{.Name}} represents a {{lower .Name}} in the system
type {{.Name}} struct {
	ID        string    ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}

// {{.Name}}Repository defines the interface for {{lower .Name}} storage
type {{.Name}}Repository interface {
	Create(ctx context.Context, m *{{.Name}}) error
	Get(ctx context.Context, id string) (*{{.Name}}, error)
	Update(ctx context.Context, m *{{.Name}}) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*{{.Name}}, error)
}

// Memory{{.Name}}Repository is an in-memory implementation of {{.Name}}Repository.
// Replace with a database-backed implementation for production.
type Memory{{.Name}}Repository struct {
	mu    sync.RWMutex
	items map[string]*{{.Name}}
	seq   int
}

func NewMemory{{.Name}}Repository() *Memory{{.Name}}Repository {
	return &Memory{{.Name}}Repository{items: make(map[string]*{{.Name}})}
}

func (r *Memory{{.Name}}Repository) Create(ctx context.Context, m *{{.Name}}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	m.ID = fmt.Sprintf("%d", r.seq)
	m.CreatedAt = time.Now()
	m.UpdatedAt = m.CreatedAt
	r.items[m.ID] = m
	return nil
}

func (r *Memory{{.Name}}Repository) Get(ctx context.Context, id string) (*{{.Name}}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.items[id]
	if !ok {
		return nil, fmt.Errorf("{{lower .Name}} %s not found", id)
	}
	return m, nil
}

func (r *Memory{{.Name}}Repository) Update(ctx context.Context, m *{{.Name}}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[m.ID]; !ok {
		return fmt.Errorf("{{lower .Name}} %s not found", m.ID)
	}
	m.UpdatedAt = time.Now()
	r.items[m.ID] = m
	return nil
}

func (r *Memory{{.Name}}Repository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return fmt.Errorf("{{lower .Name}} %s not found", id)
	}
	delete(r.items, id)
	return nil
}

func (r *Memory{{.Name}}Repository) List(ctx context.Context, offset, limit int) ([]*{{.Name}}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*{{.Name}}
	i := 0
	for _, m := range r.items {
		if i < offset {
			i++
			continue
		}
		if limit > 0 && len(result) >= limit {
			break
		}
		result = append(result, m)
		i++
	}
	return result, nil
}
`

type handlerData struct {
	Name    string
	Methods []methodData
}

type methodData struct {
	Name         string
	RequestType  string
	ResponseType string
}

type endpointData struct {
	Name   string
	Method string
	Path   string
}

type modelData struct {
	Name string
}

func generateHandler(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("handler name required: micro generate handler <name>")
	}

	name = strings.Title(strings.ToLower(name))

	// Parse methods if provided
	methods := []methodData{}
	for _, m := range c.StringSlice("method") {
		methods = append(methods, methodData{
			Name:         strings.Title(m),
			RequestType:  strings.Title(m) + "Request",
			ResponseType: strings.Title(m) + "Response",
		})
	}

	if len(methods) == 0 {
		methods = []methodData{
			{Name: "Handle", RequestType: "Request", ResponseType: "Response"},
		}
	}

	data := handlerData{
		Name:    name,
		Methods: methods,
	}

	return generateFile("handler", strings.ToLower(name)+".go", handlerTemplate, data)
}

func generateEndpoint(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("endpoint name required: micro generate endpoint <name>")
	}

	data := endpointData{
		Name:   strings.Title(strings.ToLower(name)),
		Method: strings.ToUpper(c.String("method")),
		Path:   c.String("path"),
	}

	if data.Path == "" {
		data.Path = strings.ToLower(name)
	}

	return generateFile("handler", strings.ToLower(name)+"_endpoint.go", endpointTemplate, data)
}

func generateModel(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("model name required: micro generate model <name>")
	}

	data := modelData{
		Name: strings.Title(strings.ToLower(name)),
	}

	return generateFile("model", strings.ToLower(name)+".go", modelTemplate, data)
}

func generateFile(dir, filename, tmplStr string, data interface{}) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	filepath := filepath.Join(dir, filename)

	// Check if file exists
	if _, err := os.Stat(filepath); err == nil {
		return fmt.Errorf("file %s already exists", filepath)
	}

	fn := template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	}

	tmpl, err := template.New("gen").Funcs(fn).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Created %s\n", filepath)
	return nil
}

func init() {
	cmd.Register(&cli.Command{
		Name:    "generate",
		Usage:   "Generate code scaffolding (like Rails generators)",
		Aliases: []string{"gen"},
		Subcommands: []*cli.Command{
			{
				Name:   "handler",
				Usage:  "Generate a handler: micro g handler <name>",
				Action: generateHandler,
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "method",
						Aliases: []string{"m"},
						Usage:   "Methods to generate (can be repeated)",
					},
				},
			},
			{
				Name:   "endpoint",
				Usage:  "Generate an HTTP endpoint: micro g endpoint <name>",
				Action: generateEndpoint,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "method",
						Aliases: []string{"m"},
						Usage:   "HTTP method (GET, POST, etc.)",
						Value:   "POST",
					},
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "URL path for the endpoint",
					},
				},
			},
			{
				Name:   "model",
				Usage:  "Generate a model: micro g model <name>",
				Action: generateModel,
			},
		},
	})
}
