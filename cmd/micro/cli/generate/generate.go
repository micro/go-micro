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
	"sync"
	"time"

	"go-micro.dev/v6/ai"

	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

// goMicroVersion is the go-micro.dev/v6 release pinned into the go.mod of
// every scaffolded project. This is the single source of truth — bump it
// here when cutting a release so generated services and agents stay in
// sync with the framework.
const goMicroVersion = "v6.0.0"

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
- Service names are lowercase, hyphenated, WITHOUT a "-service" suffix (e.g. "task" not "task-service", "shipping" not "shipping-service")
- Each service MUST have CRUD endpoints: Create, Read, Update, Delete, List
- Add 1-3 custom endpoints for real business logic (e.g. PlaceOrder, CheckInventory)
- Field types: string, int64, bool, float64
- Every service needs id (string), created (int64), updated (int64) fields
- Endpoint names are PascalCase
- Examples should be realistic JSON
- 2-4 services max, focused on the domain
- Keep services small and focused — one concern per service, max 5-8 fields
- Services don't call each other; an AI agent orchestrates across them`

const designPromptWithExisting = `You are a Go microservices architect using the go-micro framework.
The user has an EXISTING system with services already running. They want to extend or modify it.

Existing services:
%s

Given the user's request, return the COMPLETE set of services (existing + new/modified).
For existing services the user hasn't asked to change, return them as-is.
For new or modified services, include the full specification.

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
- Service names are lowercase, hyphenated, WITHOUT a "-service" suffix (e.g. "task" not "task-service", "shipping" not "shipping-service")
- Each service MUST have CRUD endpoints: Create, Read, Update, Delete, List
- Add custom endpoints for real business logic
- Field types: string, int64, bool, float64
- Every service needs id (string), created (int64), updated (int64) fields
- Endpoint names are PascalCase
- Examples should be realistic JSON
- Keep existing services unless the user explicitly asks to change them`

const handlerPrompt = `You are a Go developer writing a handler for a go-micro service.
Generate a COMPLETE, COMPILABLE Go handler file.

The handler must:
1. Use package "handler"
2. Import the proto package as: pb "%s/proto"
3. Import go-micro logger as: log "go-micro.dev/v6/logger"
4. Import "github.com/google/uuid" for ID generation
5. Use "go-micro.dev/v6/store" for persistent storage (NOT in-memory maps)
6. Include REAL business logic — not just CRUD store operations
7. Every exported method must have a doc comment explaining what it does
8. Every method must have an @example tag with realistic JSON input
9. Handle edge cases, validation, and return meaningful errors
10. Keep the file under 200 lines — be concise, no boilerplate

For storage, use the go-micro store package:
  import "go-micro.dev/v6/store"
  import "encoding/json"

  // In the struct:
  store store.Store

  // In the constructor:
  func New() *%s { return &%s{store: store.DefaultStore} }

  // Write a record:
  data, _ := json.Marshal(record)
  store.Write(&store.Record{Key: "prefix/" + id, Value: data})

  // Read a record:
  recs, err := store.Read("prefix/" + id)
  json.Unmarshal(recs[0].Value, &record)

  // List keys:
  keys, _ := store.List(store.ListPrefix("prefix/"))

  // Delete:
  store.Delete("prefix/" + id)

Do NOT use sync.Mutex or in-memory maps. Use store for all data.

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
// If baseDir contains existing services, they are included as context
// so the LLM extends the system rather than redesigning from scratch.
func Design(ctx context.Context, provider, apiKey, model, baseDir, prompt string) (*ServiceDesign, error) {
	m := newModel(provider, apiKey, model)
	if m == nil {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	existing := discoverExisting(baseDir)

	var sysPrompt, userPrompt string
	if len(existing) > 0 {
		sysPrompt = fmt.Sprintf(designPromptWithExisting, existing)
		userPrompt = fmt.Sprintf("Extend or modify the system: %s", prompt)
	} else {
		sysPrompt = designPrompt
		userPrompt = fmt.Sprintf("Design a microservices system for: %s", prompt)
	}

	sp := startSpinner("designing services...")
	designCtx, designCancel := context.WithTimeout(ctx, 60*time.Second)
	defer designCancel()
	resp, err := m.Generate(designCtx, &ai.Request{
		Prompt:       userPrompt,
		SystemPrompt: sysPrompt,
	})
	sp.Stop()
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

// discoverExisting scans a directory for existing go-micro services
// and returns a summary string for inclusion in the design prompt.
func discoverExisting(baseDir string) string {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}

	var summaries []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		svcDir := filepath.Join(baseDir, e.Name())

		// Look for proto files as indicator of a go-micro service
		protoDir := filepath.Join(svcDir, "proto")
		protos, err := filepath.Glob(filepath.Join(protoDir, "*.proto"))
		if err != nil || len(protos) == 0 {
			continue
		}

		proto := readFile(protos[0])
		if proto == "" {
			continue
		}

		summaries = append(summaries, fmt.Sprintf("### %s\nProto:\n```\n%s\n```", e.Name(), proto))
	}

	return strings.Join(summaries, "\n\n")
}

// Generate creates go-micro service directories from a design.
// If a service directory already exists, it skips structure generation
// but regenerates the handler (allowing iterative improvement).
func Generate(ctx context.Context, baseDir string, design *ServiceDesign, provider, apiKey, model string) error {
	m := newModel(provider, apiKey, model)

	for i, svc := range design.Services {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		svcDir := filepath.Join(baseDir, svc.Name)
		handlerFile := filepath.Join(svcDir, "handler", svc.Name+".go")
		protoFile := filepath.Join(svcDir, "proto", svc.Name+".proto")

		// Snapshot proto hash before structure generation
		protoBefore := fileHash(protoFile)

		fmt.Printf("    \033[2m[%d/%d]\033[0m generating \033[36m%s\033[0m...\n", i+1, len(design.Services), svc.Name)

		// Step 1: Generate proto (deterministic — from design spec)
		if err := generateStructure(svcDir, svc); err != nil {
			return fmt.Errorf("structure %s: %w", svc.Name, err)
		}

		protoAfter := fileHash(protoFile)
		protoChanged := protoBefore != protoAfter

		// If proto unchanged and handler unmodified, nothing to do
		if !protoChanged && protoBefore != "" && !handlerModified(svcDir, handlerFile) {
			fmt.Printf("    \033[32m✓\033[0m %s \033[2m(unchanged)\033[0m\n", svc.Name)
			continue
		}

		// Step 2: Run go mod tidy + make proto to get compiled proto
		runIn(svcDir, "go", "mod", "tidy")
		runIn(svcDir, "make", "proto")

		// Step 3: Generate handler with business logic (LLM)
		proto := readFile(protoFile)
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
		recordHandlerHash(svcDir, handlerFile)
	}

	// Generate an agent that manages all the services
	var svcNames []string
	for _, svc := range design.Services {
		svcNames = append(svcNames, svc.Name)
	}
	if err := generateAgent(baseDir, design, svcNames); err != nil {
		fmt.Printf("    \033[33m⚠\033[0m agent generation failed: %v\n", err)
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

	// Regenerate proto unless user has modified it
	protoPath := filepath.Join(dir, "proto", name+".proto")
	if !fileModified(dir, "proto_hash", protoPath) {
		writeFile(protoPath, buildProto(dehyphen, titleName, svc))
		recordFileHash(dir, "proto_hash", protoPath)
	} else {
		fmt.Printf("    \033[2mkeeping %s proto (modified)\033[0m\n", name)
	}

	// Only write structural files if directory is new
	if !exists {
		writeFile(filepath.Join(dir, "main.go"), buildMain(name, titleName))

		writeFile(filepath.Join(dir, "Makefile"),
			"GOPATH:=$(shell go env GOPATH)\n\n.PHONY: proto\nproto:\n\tprotoc --proto_path=. --micro_out=. --go_out=. proto/*.proto\n")

		writeFile(filepath.Join(dir, "go.mod"),
			fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire go-micro.dev/v6 %s\n", name, goMicroVersion))

		writeFile(filepath.Join(dir, ".gitignore"),
			fmt.Sprintf("%s\n.micro\n", name))
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
		svc.Name, titleName, titleName, titleName, titleName, proto, strings.Join(epDescs, "\n"))

	sp := startSpinner(fmt.Sprintf("writing %s handler...", svc.Name))
	genCtx, genCancel := context.WithTimeout(ctx, 90*time.Second)
	defer genCancel()
	resp, err := m.Generate(genCtx, &ai.Request{
		Prompt:       fmt.Sprintf("Generate the handler for the %s service with real business logic.", svc.Name),
		SystemPrompt: prompt,
	})
	sp.Stop()
	if err != nil {
		return err
	}

	code := firstNonEmpty(resp.Answer, resp.Reply)
	code = extractCode(code)

	if !strings.HasPrefix(strings.TrimSpace(code), "package") {
		return fmt.Errorf("LLM did not return valid Go code")
	}

	if isTruncated(code) {
		fmt.Printf("    \033[33m→\033[0m response truncated, retrying...\n")
		sp = startSpinner(fmt.Sprintf("rewriting %s handler...", svc.Name))
		retryCtx, retryCancel := context.WithTimeout(ctx, 90*time.Second)
		defer retryCancel()
		resp, err = m.Generate(retryCtx, &ai.Request{
			Prompt:       fmt.Sprintf("Generate the handler for the %s service with real business logic. Keep it concise — no more than 200 lines.", svc.Name),
			SystemPrompt: prompt,
		})
		sp.Stop()
		if err != nil {
			return err
		}
		code = firstNonEmpty(resp.Answer, resp.Reply)
		code = extractCode(code)
	}

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

		sp := startSpinner(fmt.Sprintf("fixing compile errors (attempt %d/%d)...", attempt+1, maxAttempts))
		fixCtx, fixCancel := context.WithTimeout(ctx, 60*time.Second)
		resp, fixErr := m.Generate(fixCtx, &ai.Request{
			Prompt: fmt.Sprintf("This Go code has compile errors. Fix ALL of them and return the COMPLETE corrected file.\n\nErrors:\n%s\n\nCode:\n%s",
				string(out), currentCode),
			SystemPrompt: "You are a Go expert. Return ONLY the corrected Go code. No markdown, no explanation. Start with 'package handler'.",
		})
		fixCancel()
		sp.Stop()
		if fixErr != nil {
			return fmt.Errorf("fix attempt failed: %w", fixErr)
		}

		fixed := firstNonEmpty(resp.Answer, resp.Reply)
		fixed = extractCode(fixed)
		if strings.HasPrefix(strings.TrimSpace(fixed), "package") && !isTruncated(fixed) {
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
			b.WriteString("message CreateRequest {\n")
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
			b.WriteString("message DeleteRequest {\n\tstring id = 1;\n}\n\nmessage DeleteResponse {\n\tbool deleted = 1;\n}\n\n")
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
	svcName := strings.TrimSuffix(name, "-service")
	return fmt.Sprintf(`package main

import (
	"%s/handler"
	pb "%s/proto"

	"go-micro.dev/v6"
	"go-micro.dev/v6/gateway/mcp"
)

func main() {
	service := micro.NewService("%s",
		mcp.WithMCP(":0"),
	)
	service.Init()
	pb.Register%sHandler(service.Server(), handler.New())
	service.Run()
}
`, name, name, svcName, titleName)
}

func generateAgent(baseDir string, design *ServiceDesign, svcNames []string) error {
	agentName := "agent"
	agentDir := filepath.Join(baseDir, agentName)

	if _, err := os.Stat(agentDir); err == nil {
		return nil // already exists
	}

	os.MkdirAll(agentDir, 0755)

	// Build a description of all services for the agent prompt
	var svcDescs []string
	for _, svc := range design.Services {
		var eps []string
		for _, ep := range svc.Endpoints {
			eps = append(eps, ep.Name)
		}
		svcDescs = append(svcDescs, fmt.Sprintf("- %s: %s (%s)", svc.Name, svc.Description, strings.Join(eps, ", ")))
	}

	prompt := fmt.Sprintf("You manage these services:\\n%s\\nUse the available tools to fulfill requests. Be helpful and concise.", strings.Join(svcDescs, "\\n"))
	quoted := strings.Join(svcNames, `", "`)

	writeFile(filepath.Join(agentDir, "main.go"), fmt.Sprintf(`package main

import (
	"os"

	"go-micro.dev/v6"
)

func main() {
	agent := micro.NewAgent("agent",
		micro.AgentServices("%s"),
		micro.AgentPrompt("%s"),
		micro.AgentProvider(os.Getenv("MICRO_AI_PROVIDER")),
		micro.AgentAPIKey(os.Getenv("MICRO_AI_API_KEY")),
	)
	agent.Init()
	agent.Run()
}
`, quoted, prompt))

	writeFile(filepath.Join(agentDir, "go.mod"),
		fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire go-micro.dev/v6 %s\n", agentName, goMicroVersion))

	runIn(agentDir, "go", "mod", "tidy")

	fmt.Printf("    \033[35m◆\033[0m agent \033[2m(manages: %s)\033[0m\n", strings.Join(svcNames, ", "))
	return nil
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

func isTruncated(code string) bool {
	trimmed := strings.TrimSpace(code)
	if len(trimmed) == 0 {
		return true
	}
	// Valid Go files end with a closing brace
	if trimmed[len(trimmed)-1] != '}' {
		return true
	}
	// Check balanced braces
	depth := 0
	for _, c := range trimmed {
		switch c {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth != 0
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

type spinner struct {
	msg  string
	stop chan struct{}
	done sync.WaitGroup
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func startSpinner(msg string) *spinner {
	s := &spinner{msg: msg, stop: make(chan struct{})}
	if !isTTY() {
		fmt.Printf("    %s\n", msg)
		return s
	}
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				fmt.Printf("\r\033[K")
				return
			case <-t.C:
				fmt.Printf("\r    %s %s", frames[i%len(frames)], msg)
				i++
			}
		}
	}()
	return s
}

func (s *spinner) Stop() {
	close(s.stop)
	s.done.Wait()
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

func fileModified(svcDir, key, path string) bool {
	meta := readMeta(svcDir)
	savedHash, ok := meta[key]
	if !ok {
		return false
	}
	return fileHash(path) != savedHash
}

func recordFileHash(svcDir, key, path string) {
	meta := readMeta(svcDir)
	meta[key] = fileHash(path)
	writeMeta(svcDir, meta)
}

func handlerModified(svcDir, handlerFile string) bool {
	return fileModified(svcDir, "handler_hash", handlerFile)
}

func recordHandlerHash(svcDir, handlerFile string) {
	recordFileHash(svcDir, "handler_hash", handlerFile)
}
