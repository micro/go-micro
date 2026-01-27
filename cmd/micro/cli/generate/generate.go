// Package generate provides code generation commands for micro
package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/genai"
)

var handlerTemplate = `package handler

import (
	"context"

	log "go-micro.dev/v5/logger"
)

type {{.Name}} struct{}

func New{{.Name}}() *{{.Name}} {
	return &{{.Name}}{}
}

{{range .Methods}}
// {{.Name}} handles {{.Name}} requests
func (h *{{$.Name}}) {{.Name}}(ctx context.Context, req *{{.RequestType}}, rsp *{{.ResponseType}}) error {
	log.Infof("Received {{$.Name}}.{{.Name}} request")
	// TODO: implement
	return nil
}
{{end}}
`

var endpointTemplate = `package handler

import (
	"context"
	"encoding/json"
	"net/http"

	log "go-micro.dev/v5/logger"
)

// {{.Name}}Request is the request for {{.Name}}
type {{.Name}}Request struct {
	// Add request fields here
}

// {{.Name}}Response is the response for {{.Name}}
type {{.Name}}Response struct {
	// Add response fields here
}

// {{.Name}} handles HTTP {{.Method}} requests to /{{.Path}}
func {{.Name}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Infof("Received {{.Name}} request")

	var req {{.Name}}Request
	if r.Method != http.MethodGet {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// TODO: implement handler logic
	_ = ctx
	_ = req

	rsp := {{.Name}}Response{}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rsp)
}
`

var modelTemplate = `package model

import (
	"context"
	"time"
)

// {{.Name}} represents a {{lower .Name}} in the system
type {{.Name}} struct {
	ID        string    ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
	// Add your fields here
}

// {{.Name}}Repository defines the interface for {{lower .Name}} storage
type {{.Name}}Repository interface {
	Create(ctx context.Context, m *{{.Name}}) error
	Get(ctx context.Context, id string) (*{{.Name}}, error)
	Update(ctx context.Context, m *{{.Name}}) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*{{.Name}}, error)
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

func generateWithAI(c *cli.Context) error {
	prompt := c.Args().First()
	if prompt == "" {
		return fmt.Errorf("description required: micro generate ai <description>")
	}

	gen := genai.DefaultGenAI
	if gen.String() == "noop" {
		return fmt.Errorf("no AI provider configured. Set OPENAI_API_KEY or GEMINI_API_KEY")
	}

	aiPrompt := fmt.Sprintf(`Generate Go code for a micro service handler based on this description: %s

Use the go-micro.dev/v5 framework. Include:
- Proper imports
- Handler struct with methods
- Context handling
- Logging with go-micro.dev/v5/logger
- Error handling

Only output the Go code, no explanations.`, prompt)

	ctx := context.Background()
	res, err := gen.Generate(ctx, aiPrompt)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	fmt.Println(res.Text)
	return nil
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
			{
				Name:   "ai",
				Usage:  "Generate code using AI: micro g ai <description>",
				Action: generateWithAI,
			},
		},
	})
}
