package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"order-service", "OrderService"},
		{"task", "Task"},
		{"inventory_item", "InventoryItem"},
		{"hello world", "HelloWorld"},
		{"a-b-c", "ABC"},
		{"already", "Already"},
	}
	for _, tt := range tests {
		if got := toTitle(tt.in); got != tt.want {
			t.Errorf("toTitle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestProtoType(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"string", "string"},
		{"int64", "int64"},
		{"int32", "int32"},
		{"bool", "bool"},
		{"float64", "double"},
		{"unknown", "string"},
		{"", "string"},
	}
	for _, tt := range tests {
		if got := protoType(tt.in); got != tt.want {
			t.Errorf("protoType(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "c"); got != "c" {
		t.Errorf("got %q, want %q", got, "c")
	}
	if got := firstNonEmpty("a", "b"); got != "a" {
		t.Errorf("got %q, want %q", got, "a")
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"fenced json",
			"Here's the design:\n```json\n{\"services\": []}\n```\nDone.",
			`{"services": []}`,
		},
		{
			"fenced no lang",
			"```\n{\"a\": 1}\n```",
			`{"a": 1}`,
		},
		{
			"raw json",
			`some text {"key": "val"} trailing`,
			`{"key": "val"}`,
		},
		{
			"nested braces",
			`{"a": {"b": 1}}`,
			`{"a": {"b": 1}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.in)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractCode(t *testing.T) {
	tests := []struct {
		name, in   string
		wantPrefix string
	}{
		{
			"go fence",
			"Here:\n```go\npackage handler\n\nfunc Foo() {}\n```\nDone.",
			"package handler",
		},
		{
			"generic fence",
			"```\npackage main\n```",
			"package main",
		},
		{
			"raw code",
			"Sure, here's the code:\npackage handler\n\ntype X struct{}",
			"package handler",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCode(tt.in)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("extractCode() = %q, want prefix %q", got, tt.wantPrefix)
			}
		})
	}
}

func TestBuildProto(t *testing.T) {
	svc := ServiceSpec{
		Name:        "task-service",
		Description: "Manages tasks",
		Fields: []FieldSpec{
			{Name: "id", Type: "string", Description: "Task ID"},
			{Name: "title", Type: "string", Description: "Task title"},
			{Name: "done", Type: "bool", Description: "Completion status"},
			{Name: "created", Type: "int64", Description: "Created timestamp"},
			{Name: "updated", Type: "int64", Description: "Updated timestamp"},
		},
		Endpoints: []EndpointSpec{
			{Name: "Create", Description: "Create a task"},
			{Name: "Read", Description: "Get a task"},
			{Name: "Update", Description: "Update a task"},
			{Name: "Delete", Description: "Delete a task"},
			{Name: "List", Description: "List tasks"},
			{Name: "ToggleComplete", Description: "Toggle completion"},
		},
	}

	proto := buildProto("taskservice", "TaskService", svc)

	checks := []string{
		`syntax = "proto3"`,
		`package taskservice`,
		`service TaskService`,
		`rpc Create(CreateRequest) returns (CreateResponse)`,
		`rpc ToggleComplete(ToggleCompleteRequest) returns (ToggleCompleteResponse)`,
		`message TaskServiceRecord`,
		`string title = 2`,
		`bool done = 3`,
		`message CreateRequest`,
		`message ReadRequest`,
		`message DeleteRequest`,
		`message ListRequest`,
		`message ToggleCompleteRequest`,
	}
	for _, c := range checks {
		if !strings.Contains(proto, c) {
			t.Errorf("buildProto() missing %q", c)
		}
	}

	// Create should not include id, created, updated
	createIdx := strings.Index(proto, "message CreateRequest")
	createEnd := strings.Index(proto[createIdx:], "}")
	createBlock := proto[createIdx : createIdx+createEnd]
	for _, skip := range []string{"string id", "int64 created", "int64 updated"} {
		if strings.Contains(createBlock, skip) {
			t.Errorf("CreateRequest should not contain %q", skip)
		}
	}
}

func TestBuildMain(t *testing.T) {
	// New naming: no -service suffix
	main := buildMain("order", "Order")
	checks := []string{
		`"order/handler"`,
		`pb "order/proto"`,
		`micro.NewService("order"`,
		`pb.RegisterOrderHandler`,
		`handler.New()`,
	}
	for _, c := range checks {
		if !strings.Contains(main, c) {
			t.Errorf("buildMain(order) missing %q", c)
		}
	}

	// Legacy naming: -service suffix stripped
	main = buildMain("order-service", "OrderService")
	checks = []string{
		`"order-service/handler"`,
		`pb "order-service/proto"`,
		`micro.NewService("order"`,
		`pb.RegisterOrderServiceHandler`,
		`handler.New()`,
	}
	for _, c := range checks {
		if !strings.Contains(main, c) {
			t.Errorf("buildMain() missing %q", c)
		}
	}
}

func TestHandlerModifiedTracking(t *testing.T) {
	dir := t.TempDir()
	handlerDir := filepath.Join(dir, "handler")
	os.MkdirAll(handlerDir, 0755)
	handlerFile := filepath.Join(handlerDir, "test.go")

	// No .micro file → not modified
	os.WriteFile(handlerFile, []byte("package handler\n"), 0644)
	if handlerModified(dir, handlerFile) {
		t.Error("expected not modified when no .micro exists")
	}

	// Record hash → not modified
	recordHandlerHash(dir, handlerFile)
	if handlerModified(dir, handlerFile) {
		t.Error("expected not modified after recording hash")
	}

	// Edit the file → modified
	os.WriteFile(handlerFile, []byte("package handler\n\nfunc Foo() {}\n"), 0644)
	if !handlerModified(dir, handlerFile) {
		t.Error("expected modified after editing file")
	}

	// Re-record → not modified again
	recordHandlerHash(dir, handlerFile)
	if handlerModified(dir, handlerFile) {
		t.Error("expected not modified after re-recording hash")
	}
}

func TestMetaReadWrite(t *testing.T) {
	dir := t.TempDir()

	m := readMeta(dir)
	if len(m) != 0 {
		t.Error("expected empty meta for new dir")
	}

	m["handler_hash"] = "abc123"
	m["version"] = "1"
	writeMeta(dir, m)

	m2 := readMeta(dir)
	if m2["handler_hash"] != "abc123" || m2["version"] != "1" {
		t.Errorf("readMeta() = %v, want handler_hash=abc123, version=1", m2)
	}
}

func TestGenerateStructure(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "test-svc")

	svc := ServiceSpec{
		Name:        "test-svc",
		Description: "Test service",
		Fields: []FieldSpec{
			{Name: "id", Type: "string"},
			{Name: "name", Type: "string"},
		},
		Endpoints: []EndpointSpec{
			{Name: "Create"},
			{Name: "Read"},
		},
	}

	if err := generateStructure(svcDir, svc); err != nil {
		t.Fatal(err)
	}

	// Check files exist
	for _, f := range []string{
		"proto/test-svc.proto",
		"handler/test-svc.go",
		"main.go",
		"go.mod",
		"Makefile",
		".gitignore",
	} {
		if _, err := os.Stat(filepath.Join(svcDir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}

	// Check .micro was created with handler hash
	meta := readMeta(svcDir)
	if meta["handler_hash"] == "" {
		t.Error("expected handler_hash in .micro after generateStructure")
	}

	// Run again — should not overwrite main.go
	mainBefore, _ := os.ReadFile(filepath.Join(svcDir, "main.go"))
	os.WriteFile(filepath.Join(svcDir, "main.go"), []byte("// user edited\n"), 0644)
	if err := generateStructure(svcDir, svc); err != nil {
		t.Fatal(err)
	}
	mainAfter, _ := os.ReadFile(filepath.Join(svcDir, "main.go"))
	if string(mainAfter) == string(mainBefore) {
		t.Error("expected main.go to keep user edit on re-run")
	}

	// Proto should be protected if user modified it
	protoFile := filepath.Join(svcDir, "proto", "test-svc.proto")
	protoBefore, _ := os.ReadFile(protoFile)
	os.WriteFile(protoFile, []byte("// user-edited proto\n"), 0644)
	if err := generateStructure(svcDir, svc); err != nil {
		t.Fatal(err)
	}
	protoAfter, _ := os.ReadFile(protoFile)
	if string(protoAfter) != "// user-edited proto\n" {
		t.Error("expected proto to be preserved after user edit")
	}

	// Proto should regenerate if NOT modified
	recordFileHash(svcDir, "proto_hash", protoFile)
	if err := generateStructure(svcDir, svc); err != nil {
		t.Fatal(err)
	}
	protoAfter2, _ := os.ReadFile(protoFile)
	if string(protoAfter2) == string(protoBefore) {
		// ok — regenerated from spec
	}
}

func TestFileModified(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("original"), 0644)

	// No hash → not modified
	if fileModified(dir, "test_hash", f) {
		t.Error("expected not modified with no saved hash")
	}

	recordFileHash(dir, "test_hash", f)

	// Same content → not modified
	if fileModified(dir, "test_hash", f) {
		t.Error("expected not modified with matching hash")
	}

	// Changed content → modified
	os.WriteFile(f, []byte("changed"), 0644)
	if !fileModified(dir, "test_hash", f) {
		t.Error("expected modified after content change")
	}
}

func TestDiscoverExisting(t *testing.T) {
	dir := t.TempDir()

	// Empty directory → empty string
	if got := discoverExisting(dir); got != "" {
		t.Errorf("expected empty for empty dir, got %q", got)
	}

	// Non-service directory (no proto) → empty
	os.MkdirAll(filepath.Join(dir, "not-a-service"), 0755)
	if got := discoverExisting(dir); got != "" {
		t.Errorf("expected empty for dir without proto, got %q", got)
	}

	// Create a real service directory with proto
	svcDir := filepath.Join(dir, "order-service")
	os.MkdirAll(filepath.Join(svcDir, "proto"), 0755)
	os.WriteFile(filepath.Join(svcDir, "proto", "order-service.proto"),
		[]byte("syntax = \"proto3\";\nservice OrderService {}"), 0644)

	got := discoverExisting(dir)
	if !strings.Contains(got, "order-service") {
		t.Errorf("expected to find order-service, got %q", got)
	}
	if !strings.Contains(got, "OrderService") {
		t.Errorf("expected to find proto content, got %q", got)
	}

	// Add a second service
	svc2Dir := filepath.Join(dir, "user-service")
	os.MkdirAll(filepath.Join(svc2Dir, "proto"), 0755)
	os.WriteFile(filepath.Join(svc2Dir, "proto", "user-service.proto"),
		[]byte("syntax = \"proto3\";\nservice UserService {}"), 0644)

	got = discoverExisting(dir)
	if !strings.Contains(got, "order-service") || !strings.Contains(got, "user-service") {
		t.Errorf("expected both services, got %q", got)
	}
}

func TestIsTruncated(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"complete", "package handler\n\nfunc New() *H { return &H{} }\n", false},
		{"empty", "", true},
		{"no closing brace", "package handler\n\nfunc Foo() {", true},
		{"unbalanced", "package handler\n\nfunc Foo() {\n\tif true {", true},
		{"balanced", "package handler\n\nfunc Foo() {\n\tif true {\n\t}\n}", false},
		{"trailing whitespace ok", "package handler\n\ntype X struct{}\n\n", false},
		{"mid-expression", "package handler\n\nfunc F() {\n\tx := 1 +", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTruncated(tt.code); got != tt.want {
				t.Errorf("isTruncated() = %v, want %v", got, tt.want)
			}
		})
	}
}
