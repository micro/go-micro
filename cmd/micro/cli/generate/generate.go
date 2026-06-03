// Package generate implements 'micro new --prompt' and 'micro run --prompt'
// which use an LLM to design services from a natural language description,
// then generate real go-micro code using the existing template system.
package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-micro.dev/v5/ai"

	// Register providers so ai.New works.
	_ "go-micro.dev/v5/ai/anthropic"
	_ "go-micro.dev/v5/ai/atlascloud"
	_ "go-micro.dev/v5/ai/gemini"
	_ "go-micro.dev/v5/ai/groq"
	_ "go-micro.dev/v5/ai/mistral"
	_ "go-micro.dev/v5/ai/openai"
	_ "go-micro.dev/v5/ai/together"
)

const designPrompt = `You are a Go microservices architect. Given a description of a system, design the services needed.

Return ONLY valid JSON with this exact structure:
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
- Service names are lowercase, hyphenated (e.g. "order-items")
- Each service should have CRUD endpoints (Create, Read, Update, Delete, List) plus any custom ones
- Field types must be: string, int64, bool, float64
- Every service must have an "id" field (string) and "created"/"updated" fields (int64)
- Endpoint names are PascalCase (e.g. "Create", "FindByEmail")
- The example field shows a realistic JSON input for the endpoint
- Keep it focused — 2-5 services max
- Each endpoint description should be clear enough for an AI agent to understand when to call it`

// ServiceDesign is the LLM's output describing the system architecture.
type ServiceDesign struct {
	Services []ServiceSpec `json:"services"`
}

// ServiceSpec describes one service to generate.
type ServiceSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Fields      []FieldSpec    `json:"fields"`
	Endpoints   []EndpointSpec `json:"endpoints"`
}

// FieldSpec describes a field on the service's record type.
type FieldSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// EndpointSpec describes an RPC endpoint.
type EndpointSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// Design calls an LLM to design services from a natural language prompt.
func Design(ctx context.Context, provider, apiKey, model, prompt string) (*ServiceDesign, error) {
	if provider == "" {
		provider = ai.AutoDetectProvider("")
	}

	var opts []ai.Option
	opts = append(opts, ai.WithAPIKey(apiKey))
	if model != "" {
		opts = append(opts, ai.WithModel(model))
	}

	m := ai.New(provider, opts...)
	if m == nil {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	resp, err := m.Generate(ctx, &ai.Request{
		Prompt:       fmt.Sprintf("Design a microservices system for: %s", prompt),
		SystemPrompt: designPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM design failed: %w", err)
	}

	reply := resp.Reply
	if resp.Answer != "" {
		reply = resp.Answer
	}

	// Extract JSON from the response (LLM may wrap it in markdown)
	reply = extractJSON(reply)

	var design ServiceDesign
	if err := json.Unmarshal([]byte(reply), &design); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nResponse: %s", err, reply)
	}

	if len(design.Services) == 0 {
		return nil, fmt.Errorf("LLM returned no services")
	}

	return &design, nil
}

// Generate creates go-micro service directories from a design.
// Each service gets proto, handler, main.go, Makefile, go.mod.
func Generate(baseDir string, design *ServiceDesign) error {
	for _, svc := range design.Services {
		svcDir := filepath.Join(baseDir, svc.Name)
		if err := generateService(svcDir, svc); err != nil {
			return fmt.Errorf("generate %s: %w", svc.Name, err)
		}
	}
	return nil
}

func generateService(dir string, svc ServiceSpec) error {
	if err := os.MkdirAll(filepath.Join(dir, "handler"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "proto"), 0755); err != nil {
		return err
	}

	name := svc.Name
	titleName := toTitle(name)

	// Proto
	proto := generateProto(name, titleName, svc)
	if err := os.WriteFile(filepath.Join(dir, "proto", name+".proto"), []byte(proto), 0644); err != nil {
		return err
	}

	// Handler
	handler := generateHandler(name, titleName, svc)
	if err := os.WriteFile(filepath.Join(dir, "handler", name+".go"), []byte(handler), 0644); err != nil {
		return err
	}

	// Main
	main := generateMain(name, titleName)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0644); err != nil {
		return err
	}

	// Makefile
	makefile := fmt.Sprintf("GOPATH:=$(shell go env GOPATH)\n\n.PHONY: proto\nproto:\n\tprotoc --proto_path=. --micro_out=. --go_out=. proto/*.proto\n")
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0644); err != nil {
		return err
	}

	// go.mod
	gomod := fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire (\n\tgo-micro.dev/v5 latest\n\tgithub.com/golang/protobuf latest\n\tgoogle.golang.org/protobuf latest\n)\n", name)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		return err
	}

	// Run go mod tidy and make proto
	runIn(dir, "go", "mod", "tidy")
	runIn(dir, "make", "proto")

	return nil
}

func generateProto(name, titleName string, svc ServiceSpec) string {
	dehyphen := strings.ReplaceAll(name, "-", "")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("syntax = \"proto3\";\n\npackage %s;\n\noption go_package = \"./proto;%s\";\n\n", dehyphen, dehyphen))

	// Service definition
	b.WriteString(fmt.Sprintf("service %s {\n", titleName))
	for _, ep := range svc.Endpoints {
		reqName := ep.Name + "Request"
		rspName := ep.Name + "Response"
		if ep.Name == "List" {
			reqName = "ListRequest"
			rspName = "ListResponse"
		}
		b.WriteString(fmt.Sprintf("\trpc %s(%s) returns (%s) {}\n", ep.Name, reqName, rspName))
	}
	b.WriteString("}\n\n")

	// Record message
	b.WriteString(fmt.Sprintf("message %sRecord {\n", titleName))
	fieldNum := 1
	for _, f := range svc.Fields {
		b.WriteString(fmt.Sprintf("\t// %s\n", f.Description))
		b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, fieldNum))
		fieldNum++
	}
	b.WriteString("}\n\n")

	// Request/response messages for each endpoint
	for _, ep := range svc.Endpoints {
		switch ep.Name {
		case "Create":
			b.WriteString(fmt.Sprintf("message CreateRequest {\n"))
			fn := 1
			for _, f := range svc.Fields {
				if f.Name == "id" || f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, fn))
				fn++
			}
			b.WriteString("}\n\n")
			b.WriteString(fmt.Sprintf("message CreateResponse {\n\t%sRecord record = 1;\n}\n\n", titleName))
		case "Read":
			b.WriteString("message ReadRequest {\n\tstring id = 1;\n}\n\n")
			b.WriteString(fmt.Sprintf("message ReadResponse {\n\t%sRecord record = 1;\n}\n\n", titleName))
		case "Update":
			b.WriteString("message UpdateRequest {\n\tstring id = 1;\n")
			fn := 2
			for _, f := range svc.Fields {
				if f.Name == "id" || f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, fn))
				fn++
			}
			b.WriteString("}\n\n")
			b.WriteString(fmt.Sprintf("message UpdateResponse {\n\t%sRecord record = 1;\n}\n\n", titleName))
		case "Delete":
			b.WriteString("message DeleteRequest {\n\tstring id = 1;\n}\n\n")
			b.WriteString("message DeleteResponse {\n\tbool deleted = 1;\n}\n\n")
		case "List":
			b.WriteString("message ListRequest {\n\tint64 limit = 1;\n\tint64 offset = 2;\n}\n\n")
			b.WriteString(fmt.Sprintf("message ListResponse {\n\trepeated %sRecord records = 1;\n\tint64 total = 2;\n}\n\n", titleName))
		default:
			// Custom endpoint — request has the fields, response has a result string
			b.WriteString(fmt.Sprintf("message %sRequest {\n", ep.Name))
			fn := 1
			for _, f := range svc.Fields {
				if f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t%s %s = %d;\n", protoType(f.Type), f.Name, fn))
				fn++
			}
			b.WriteString("}\n\n")
			b.WriteString(fmt.Sprintf("message %sResponse {\n\tstring result = 1;\n}\n\n", ep.Name))
		}
	}

	return b.String()
}

func generateHandler(name, titleName string, svc ServiceSpec) string {
	dehyphen := strings.ReplaceAll(name, "-", "")
	var b strings.Builder

	b.WriteString("package handler\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"sort\"\n\t\"sync\"\n\t\"time\"\n\n")
	b.WriteString("\t\"github.com/google/uuid\"\n")
	b.WriteString(fmt.Sprintf("\tlog \"go-micro.dev/v5/logger\"\n\n\tpb \"%s/proto\"\n)\n\n", name))

	b.WriteString(fmt.Sprintf("type %s struct {\n\tmu      sync.RWMutex\n\trecords map[string]*pb.%sRecord\n}\n\n", titleName, titleName))
	b.WriteString(fmt.Sprintf("func New() *%s {\n\treturn &%s{records: make(map[string]*pb.%sRecord)}\n}\n\n", titleName, titleName, titleName))

	for _, ep := range svc.Endpoints {
		example := ep.Example
		if example == "" {
			example = "{}"
		}
		b.WriteString(fmt.Sprintf("// %s %s\n//\n// @example %s\n", ep.Name, ep.Description, example))

		switch ep.Name {
		case "Create":
			b.WriteString(fmt.Sprintf("func (h *%s) Create(ctx context.Context, req *pb.CreateRequest, rsp *pb.CreateResponse) error {\n", titleName))
			b.WriteString(fmt.Sprintf("\tlog.Infof(\"Creating %s record\")\n", dehyphen))
			b.WriteString("\tnow := time.Now().Unix()\n")
			b.WriteString(fmt.Sprintf("\trecord := &pb.%sRecord{\n\t\tId: uuid.New().String(),\n", titleName))
			for _, f := range svc.Fields {
				if f.Name == "id" || f.Name == "created" || f.Name == "updated" {
					continue
				}
				b.WriteString(fmt.Sprintf("\t\t%s: req.%s,\n", toTitle(f.Name), toTitle(f.Name)))
			}
			b.WriteString("\t\tCreated: now,\n\t\tUpdated: now,\n\t}\n")
			b.WriteString("\th.mu.Lock()\n\th.records[record.Id] = record\n\th.mu.Unlock()\n\trsp.Record = record\n\treturn nil\n}\n\n")

		case "Read":
			b.WriteString(fmt.Sprintf("func (h *%s) Read(ctx context.Context, req *pb.ReadRequest, rsp *pb.ReadResponse) error {\n", titleName))
			b.WriteString("\th.mu.RLock()\n\trecord, ok := h.records[req.Id]\n\th.mu.RUnlock()\n")
			b.WriteString("\tif !ok {\n\t\treturn fmt.Errorf(\"record %s not found\", req.Id)\n\t}\n")
			b.WriteString("\trsp.Record = record\n\treturn nil\n}\n\n")

		case "Update":
			b.WriteString(fmt.Sprintf("func (h *%s) Update(ctx context.Context, req *pb.UpdateRequest, rsp *pb.UpdateResponse) error {\n", titleName))
			b.WriteString("\th.mu.Lock()\n\tdefer h.mu.Unlock()\n")
			b.WriteString("\trecord, ok := h.records[req.Id]\n")
			b.WriteString("\tif !ok {\n\t\treturn fmt.Errorf(\"record %s not found\", req.Id)\n\t}\n")
			for _, f := range svc.Fields {
				if f.Name == "id" || f.Name == "created" || f.Name == "updated" {
					continue
				}
				switch f.Type {
				case "string":
					b.WriteString(fmt.Sprintf("\tif req.%s != \"\" {\n\t\trecord.%s = req.%s\n\t}\n", toTitle(f.Name), toTitle(f.Name), toTitle(f.Name)))
				case "int64":
					b.WriteString(fmt.Sprintf("\tif req.%s != 0 {\n\t\trecord.%s = req.%s\n\t}\n", toTitle(f.Name), toTitle(f.Name), toTitle(f.Name)))
				case "bool":
					b.WriteString(fmt.Sprintf("\trecord.%s = req.%s\n", toTitle(f.Name), toTitle(f.Name)))
				default:
					b.WriteString(fmt.Sprintf("\trecord.%s = req.%s\n", toTitle(f.Name), toTitle(f.Name)))
				}
			}
			b.WriteString("\trecord.Updated = time.Now().Unix()\n\trsp.Record = record\n\treturn nil\n}\n\n")

		case "Delete":
			b.WriteString(fmt.Sprintf("func (h *%s) Delete(ctx context.Context, req *pb.DeleteRequest, rsp *pb.DeleteResponse) error {\n", titleName))
			b.WriteString("\th.mu.Lock()\n\t_, ok := h.records[req.Id]\n\tif ok {\n\t\tdelete(h.records, req.Id)\n\t}\n\th.mu.Unlock()\n")
			b.WriteString("\trsp.Deleted = ok\n\treturn nil\n}\n\n")

		case "List":
			b.WriteString(fmt.Sprintf("func (h *%s) List(ctx context.Context, req *pb.ListRequest, rsp *pb.ListResponse) error {\n", titleName))
			b.WriteString(fmt.Sprintf("\th.mu.RLock()\n\tdefer h.mu.RUnlock()\n\tall := make([]*pb.%sRecord, 0, len(h.records))\n", titleName))
			b.WriteString("\tfor _, r := range h.records {\n\t\tall = append(all, r)\n\t}\n")
			b.WriteString("\tsort.Slice(all, func(i, j int) bool { return all[i].Created > all[j].Created })\n")
			b.WriteString("\trsp.Total = int64(len(all))\n")
			b.WriteString("\toffset := int(req.Offset)\n\tif offset > len(all) { offset = len(all) }\n")
			b.WriteString("\tlimit := int(req.Limit)\n\tif limit <= 0 { limit = 20 }\n")
			b.WriteString("\tend := offset + limit\n\tif end > len(all) { end = len(all) }\n")
			b.WriteString("\trsp.Records = all[offset:end]\n\treturn nil\n}\n\n")

		default:
			// Custom endpoint — simple stub
			b.WriteString(fmt.Sprintf("func (h *%s) %s(ctx context.Context, req *pb.%sRequest, rsp *pb.%sResponse) error {\n",
				titleName, ep.Name, ep.Name, ep.Name))
			b.WriteString(fmt.Sprintf("\tlog.Infof(\"%s.%s called\")\n", titleName, ep.Name))
			b.WriteString("\trsp.Result = \"ok\"\n\treturn nil\n}\n\n")
		}
	}

	// Suppress unused imports
	b.WriteString("var _ = fmt.Sprintf\nvar _ = sort.Slice\n")

	return b.String()
}

func generateMain(name, titleName string) string {
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
	// Try to find JSON in markdown code blocks
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
	// Try to find raw JSON
	if i := strings.Index(s, "{"); i >= 0 {
		depth := 0
		for j := i; j < len(s); j++ {
			if s[j] == '{' {
				depth++
			} else if s[j] == '}' {
				depth--
				if depth == 0 {
					return s[i : j+1]
				}
			}
		}
	}
	return s
}

func protoType(goType string) string {
	switch goType {
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
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

func runIn(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
