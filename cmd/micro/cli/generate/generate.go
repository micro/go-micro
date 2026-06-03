// Package generate implements AI-powered service generation for go-micro.
// It uses an LLM to design service architecture and generate handler code
// with real business logic, then compiles and fixes errors iteratively.
package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-micro.dev/v5/ai"

	_ "go-micro.dev/v5/ai/anthropic"
	_ "go-micro.dev/v5/ai/atlascloud"
	_ "go-micro.dev/v5/ai/gemini"
	_ "go-micro.dev/v5/ai/groq"
	_ "go-micro.dev/v5/ai/mistral"
	_ "go-micro.dev/v5/ai/openai"
	_ "go-micro.dev/v5/ai/together"
)

const designPrompt = `You are a Go microservices architect using the go-micro framework.
Given a system description, design the services needed.

Return ONLY valid JSON:
{
  "services": [
    {
      "name": "service-name",
      "description": "What this service does",
      "fields": [
        {"name": "field_name", "type": "string", "description": "What this field is"}
      ],
      "endpoints": [
        {"name": "EndpointName", "description": "What this endpoint does", "example": "{\"key\": \"value\"}"}
      ]
    }
  ]
}

Rules:
- Service names are lowercase, hyphenated
- Each service MUST have CRUD endpoints: Create, Read, Update, Delete, List
- Add custom endpoints for real business logic (e.g. PlaceOrder, CheckInventory, CalculateShipping)
- Field types: string, int64, bool, float64
- Every service needs id (string), created (int64), updated (int64) fields
- Endpoint names are PascalCase
- Examples should be realistic JSON
- 2-5 services max, focused on the domain`

const handlerPrompt = `You are a Go developer writing a handler for a go-micro service.
Generate a COMPLETE, COMPILABLE Go handler file.

The handler must:
1. Use package "handler"
2. Import the proto package as: pb "%s/proto"
3. Import go-micro logger as: log "go-micro.dev/v5/logger"
4. Import "github.com/google/uuid" for ID generation
5. Use sync.RWMutex for thread-safe in-memory storage
6. Include REAL business logic — not just CRUD map operations
7. Every exported method must have a doc comment explaining what it does
8. Every method must have an @example tag with realistic JSON input
9. Handle edge cases, validation, and return meaningful errors

The struct name is %s.
The constructor is func New() *%s.

Here is the proto definition:
%s

Here is what each endpoint should do:
%s

Return ONLY the Go code. No markdown, no explanation. Just the .go file content starting with "package handler".`

// ServiceDesign is the LLM's output.
type ServiceDesign struct {
	Services []ServiceSpec `json:"services"`
}

type ServiceSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Fields      []FieldSpec    `json:"fields"`
	Endpoints   []EndpointSpec `json:"endpoints"`
}

type FieldSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type EndpointSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// Design calls an LLM to design services from a prompt.
func Design(ctx context.Context, provider, apiKey, model, prompt string) (*ServiceDesign, error) {
	m := newModel(provider, apiKey, model)
	if m == nil {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	resp, err := m.Generate(ctx, &ai.Request{
		Prompt:       fmt.Sprintf("Design a microservices system for: %s", prompt),
		SystemPrompt: designPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("design failed: %w", err)
	}

	reply := firstNonEmpty(resp.Answer, resp.Reply)
	reply = extractJSON(reply)

	var design ServiceDesign
	if err := json.Unmarshal([]byte(reply), &design); err != nil {
		return nil, fmt.Errorf("failed to parse design: %w\nResponse: %s", err, reply)
	}
	if len(design.Services) == 0 {
		return nil, fmt.Errorf("no services designed")
	}
	return &design, nil
}

// Generate creates go-micro service directories from a design.
// If a service directory already exists, it skips structure generation
// but regenerates the handler (allowing iterative improvement).
func Generate(ctx context.Context, baseDir string, design *ServiceDesign, provider, apiKey, model string) error {
	m := newModel(provider, apiKey, model)

	for i, svc := range design.Services {
		svcDir := filepath.Join(baseDir, svc.Name)
		fmt.Printf("    \033[2m[%d/%d]\033[0m generating \033[36m%s\033[0m...\n", i+1, len(design.Services), svc.Name)

		// Step 1: Generate proto (deterministic — from design spec)
		if err := generateStructure(svcDir, svc); err != nil {
			return fmt.Errorf("structure %s: %w", svc.Name, err)
		}

		// Step 2: Run go mod tidy + make proto to get compiled proto
		runIn(svcDir, "go", "mod", "tidy")
		runIn(svcDir, "make", "proto")

		// Step 3: Generate handler with business logic (LLM)
		proto := readFile(filepath.Join(svcDir, "proto", svc.Name+".proto"))
		if err := generateHandler(ctx, m, svcDir, svc, proto); err != nil {
			return fmt.Errorf("handler %s: %w", svc.Name, err)
		}

		// Step 4: Compile-fix loop
		if err := compileFix(ctx, m, svcDir, svc.Name, 3); err != nil {
			fmt.Printf("    \033[33m⚠\033[0m %s has compile errors (may need manual fix)\n", svc.Name)
		} else {
			fmt.Printf("    \033[32m✓\033[0m %s\n", svc.Name)
		}

		// Record final handler hash (after any compile fixes)
		recordHandlerHash(svcDir, filepath.Join(svcDir, "handler", svc.Name+".go"))
	}
	return nil
}

// generateStructure creates the proto, main.go, go.mod, Makefile.
// If the directory already exists, only regenerates the proto
// (handler will be regenerated separately by the LLM).
func generateStructure(dir string, svc ServiceSpec) error {
	exists := false
	if _, err := os.Stat(dir); err == nil {
		exists = true
	}
	os.MkdirAll(filepath.Join(dir, "handler"), 0755)
	os.MkdirAll(filepath.Join(dir, "proto"), 0755)

	name := svc.Name
	titleName := toTitle(name)
	dehyphen := strings.ReplaceAll(name, "-", "")

	// Always regenerate proto (design may have changed)
	writeFile(filepath.Join(dir, "proto", name+".proto"), buildProto(dehyphen, titleName, svc))

	// Only write structural files if directory is new
	if !exists {
		writeFile(filepath.Join(dir, "main.go"), buildMain(name, titleName))

		writeFile(filepath.Join(dir, "Makefile"),
			"GOPATH:=$(shell go env GOPATH)\n\n.PHONY: proto\nproto:\n\tprotoc --proto_path=. --micro_out=. --go_out=. proto/*.proto\n")

		writeFile(filepath.Join(dir, "go.mod"),
			fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire (\n\tgo-micro.dev/v5 latest\n\tgithub.com/golang/protobuf latest\n\tgoogle.golang.org/protobuf latest\n)\n", name))
	}

	// Placeholder handler so go mod tidy works (will be overwritten by LLM)
	handlerPath := filepath.Join(dir, "handler", name+".go")
	if _, err := os.Stat(handlerPath); os.IsNotExist(err) {
		writeFile(handlerPath,
			fmt.Sprintf("package handler\n\ntype %s struct{}\n\nfunc New() *%s { return &%s{} }\n", titleName, titleName, titleName))
		recordHandlerHash(dir, handlerPath)
	}

	return nil
}

// generateHandler asks the LLM to write the handler with business logic.
// If the handler exists and the user has modified it since generation,
// it is left untouched.
func generateHandler(ctx context.Context, m ai.Model, dir string, svc ServiceSpec, proto string) error {
	if m == nil {
		return nil // no LLM — keep the placeholder
	}

	handlerFile := filepath.Join(dir, "handler", svc.Name+".go")

	if handlerModified(dir, handlerFile) {
		fmt.Printf("    \033[2mkeeping %s handler (modified)\033[0m\n", svc.Name)
		return nil
	}

	titleName := toTitle(svc.Name)

	// Build endpoint descriptions
	var epDescs []string
	for _, ep := range svc.Endpoints {
		epDescs = append(epDescs, fmt.Sprintf("- %s: %s (example input: %s)", ep.Name, ep.Description, ep.Example))
	}

	prompt := fmt.Sprintf(handlerPrompt,
		svc.Name, titleName, titleName, proto, strings.Join(epDescs, "\n"))

	resp, err := m.Generate(ctx, &ai.Request{
		Prompt:       fmt.Sprintf("Generate the handler for the %s service with real business logic.", svc.Name),
		SystemPrompt: prompt,
	})
	if err != nil {
		return err
	}

	code := firstNonEmpty(resp.Answer, resp.Reply)
	code = extractCode(code)

	if !strings.HasPrefix(strings.TrimSpace(code), "package") {
		return fmt.Errorf("LLM did not return valid Go code")
	}

	writeFile(handlerFile, code)
	recordHandlerHash(dir, handlerFile)
	return nil
}

// compileFix tries to compile, and if it fails, sends the error to
// the LLM to fix. Up to maxAttempts iterations.
func compileFix(ctx context.Context, m ai.Model, dir, name string, maxAttempts int) error {
	for attempt := 0; attempt < maxAttempts; attempt++ {
		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil // compiles!
		}

		if m == nil {
			return fmt.Errorf("compile failed: %s", string(out))
		}

		// Read current handler
		handlerPath := filepath.Join(dir, "handler", name+".go")
		currentCode := readFile(handlerPath)

		fmt.Printf("    \033[33m→\033[0m compile error, fixing (attempt %d/%d)...\n", attempt+1, maxAttempts)

		resp, fixErr := m.Generate(ctx, &ai.Request{
			Prompt: fmt.Sprintf("This Go code has compile errors. Fix ALL of them and return the COMPLETE corrected file.\n\nErrors:\n%s\n\nCode:\n%s",
				string(out), currentCode),
			SystemPrompt: "You are a Go expert. Return ONLY the corrected Go code. No markdown, no explanation. Start with 'package handler'.",
		})
		if fixErr != nil {
			return fmt.Errorf("fix attempt failed: %w", fixErr)
		}

		fixed := firstNonEmpty(resp.Answer, resp.Reply)
		fixed = extractCode(fixed)
		if strings.HasPrefix(strings.TrimSpace(fixed), "package") {
			writeFile(handlerPath, fixed)
		}
	}

	// Final check
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("still fails after %d attempts: %s", maxAttempts, string(out))
	}
	return nil
}

func newModel(provider, apiKey, model string) ai.Model {
	if provider == "" {
		provider = ai.AutoDetectProvider("")
	}
	var opts []ai.Option
	opts = append(opts, ai.WithAPIKey(apiKey))
	if model != "" {
		opts = append(opts, ai.WithModel(model))
	}
	return ai.New(provider, opts...)
}

func buildProto(dehyphen, titleName string, svc ServiceSpec) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("syntax = \"proto3\";\n\npackage %s;\n\noption go_package = \"./proto;%s\";\n\n", dehyphen, dehyphen))

	b.WriteString(fmt.Sprintf("service %s {\n", titleName))
	for _, ep := range svc.Endpoints {
		b.WriteString(fmt.Sprintf("\trpc %s(%sRequest) returns (%sResponse) {}\n", ep.Name, ep.Name, ep.Name))
	}
	b.WriteString("}\n\n")

	// Record message
	b.WriteString(fmt.Sprintf("message %sRecord {\n", titleName))
	for i, f := range svc.Fields {
		b.WriteString(fmt.Sprintf("\t%s %s = %d; // %s\n", protoType(f.Type), f.Name, i+1, f.Description))
	}
	b.WriteString("}\n\n")

	// Request/response for each endpoint
	for _, ep := range svc.Endpoints {
		switch ep.Name {
		case "Create":
			b.WriteString(fmt.Sprintf("message CreateRequest {\n"))
			n := 1
			for _, f := range svc.Fields {
				if f.Name == "id" || f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, n))
				n++
			}
			b.WriteString(fmt.Sprintf("}\n\nmessage CreateResponse {\n\t%sRecord record = 1;\n}\n\n", titleName))
		case "Read":
			b.WriteString(fmt.Sprintf("message ReadRequest {\n\tstring id = 1;\n}\n\nmessage ReadResponse {\n\t%sRecord record = 1;\n}\n\n", titleName))
		case "Update":
			b.WriteString("message UpdateRequest {\n\tstring id = 1;\n")
			n := 2
			for _, f := range svc.Fields {
				if f.Name == "id" || f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, n))
				n++
			}
			b.WriteString(fmt.Sprintf("}\n\nmessage UpdateResponse {\n\t%sRecord record = 1;\n}\n\n", titleName))
		case "Delete":
			b.WriteString(fmt.Sprintf("message DeleteRequest {\n\tstring id = 1;\n}\n\nmessage DeleteResponse {\n\tbool deleted = 1;\n}\n\n"))
		case "List":
			b.WriteString(fmt.Sprintf("message ListRequest {\n\tint64 limit = 1;\n\tint64 offset = 2;\n\tstring query = 3;\n}\n\nmessage ListResponse {\n\trepeated %sRecord records = 1;\n\tint64 total = 2;\n}\n\n", titleName))
		default:
			// Custom endpoint — use all fields as input, record as output
			b.WriteString(fmt.Sprintf("message %sRequest {\n", ep.Name))
			n := 1
			for _, f := range svc.Fields {
				if f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, n))
				n++
			}
			b.WriteString(fmt.Sprintf("}\n\nmessage %sResponse {\n\t%sRecord record = 1;\n\tstring message = 2;\n\tbool success = 3;\n}\n\n", ep.Name, titleName))
		}
	}
	return b.String()
}

func buildMain(name, titleName string) string {
	return fmt.Sprintf(`package main

import (
	"%s/handler"
	pb "%s/proto"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
)

func main() {
	service := micro.New("%s",
		mcp.WithMCP(":3001"),
	)
	service.Init()
	pb.Register%sHandler(service.Server(), handler.New())
	service.Run()
}
`, name, name, strings.ReplaceAll(name, "-", ""), titleName)
}

func extractJSON(s string) string {
	if i := strings.Index(s, "```json"); i >= 0 {
		s = s[i+7:]
		if j := strings.Index(s, "```"); j >= 0 {
			return strings.TrimSpace(s[:j])
		}
	}
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		if j := strings.Index(s, "```"); j >= 0 {
			return strings.TrimSpace(s[:j])
		}
	}
	if i := strings.Index(s, "{"); i >= 0 {
		depth := 0
		for j := i; j < len(s); j++ {
			switch s[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return s[i : j+1]
				}
			}
		}
	}
	return s
}

func extractCode(s string) string {
	if i := strings.Index(s, "```go"); i >= 0 {
		s = s[i+5:]
		if j := strings.Index(s, "```"); j >= 0 {
			return strings.TrimSpace(s[:j])
		}
	}
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		if j := strings.Index(s, "```"); j >= 0 {
			return strings.TrimSpace(s[:j])
		}
	}
	// Try to find raw package declaration
	if i := strings.Index(s, "package "); i >= 0 {
		return strings.TrimSpace(s[i:])
	}
	return strings.TrimSpace(s)
}

func protoType(t string) string {
	switch t {
	case "int64":
		return "int64"
	case "int32":
		return "int32"
	case "bool":
		return "bool"
	case "float64":
		return "double"
	default:
		return "string"
	}
}

func toTitle(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func readFile(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

func writeFile(path, content string) {
	os.WriteFile(path, []byte(content), 0644)
}

func runIn(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH")+":"+os.Getenv("GOPATH")+"/bin:"+os.Getenv("HOME")+"/go/bin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fileHash(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func metaPath(svcDir string) string {
	return filepath.Join(svcDir, ".micro")
}

func readMeta(svcDir string) map[string]string {
	m := make(map[string]string)
	b, err := os.ReadFile(metaPath(svcDir))
	if err != nil {
		return m
	}
	json.Unmarshal(b, &m)
	return m
}

func writeMeta(svcDir string, m map[string]string) {
	b, _ := json.MarshalIndent(m, "", "  ")
	os.WriteFile(metaPath(svcDir), b, 0644)
}

func handlerModified(svcDir, handlerFile string) bool {
	meta := readMeta(svcDir)
	savedHash, ok := meta["handler_hash"]
	if !ok {
		return false // no hash recorded — treat as unmodified (first run or pre-tracking)
	}
	return fileHash(handlerFile) != savedHash
}

func recordHandlerHash(svcDir, handlerFile string) {
	meta := readMeta(svcDir)
	meta["handler_hash"] = fileHash(handlerFile)
	writeMeta(svcDir, meta)
}
