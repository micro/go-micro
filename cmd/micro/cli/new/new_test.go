package new

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	tmpl "go-micro.dev/v5/cmd/micro/cli/new/template"
)

func TestTemplatesParse(t *testing.T) {
	fn := template.FuncMap{
		"title": func(s string) string {
			return strings.ReplaceAll(strings.Title(s), "-", "")
		},
		"dehyphen": func(s string) string {
			return strings.ReplaceAll(s, "-", "")
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	templates := map[string]string{
		"SimpleMain":       tmpl.SimpleMain,
		"SimpleMainMCP":    tmpl.SimpleMainMCP,
		"SimpleMakefile":   tmpl.SimpleMakefile,
		"SimpleModule":     tmpl.SimpleModule,
		"SimpleReadme":     tmpl.SimpleReadme,
		"SimpleReadmeMCP":  tmpl.SimpleReadmeMCP,
		"MainSRV":          tmpl.MainSRV,
		"MainSRVNoMCP":     tmpl.MainSRVNoMCP,
		"HandlerSRV":       tmpl.HandlerSRV,
		"ProtoSRV":         tmpl.ProtoSRV,
		"Makefile":         tmpl.Makefile,
		"Module":           tmpl.Module,
		"Readme":           tmpl.Readme,
		"GitIgnore":        tmpl.GitIgnore,
	}

	data := config{
		Alias:  "testservice",
		Dir:    "testservice",
		GoDir:  "/tmp/test",
		GoPath: "/tmp",
	}

	for name, src := range templates {
		t.Run(name, func(t *testing.T) {
			tmplObj, err := template.New(name).Funcs(fn).Parse(src)
			if err != nil {
				t.Fatalf("failed to parse template %s: %v", name, err)
			}

			var buf strings.Builder
			if err := tmplObj.Execute(&buf, data); err != nil {
				t.Fatalf("failed to execute template %s: %v", name, err)
			}

			if buf.Len() == 0 {
				t.Fatalf("template %s produced empty output", name)
			}
		})
	}
}

func TestCreateSimpleService(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "mysvc")

	c := config{
		Alias:  "mysvc",
		Dir:    svcDir,
		GoDir:  svcDir,
		GoPath: dir,
		Files: []file{
			{"main.go", tmpl.SimpleMainMCP},
			{"Makefile", tmpl.SimpleMakefile},
			{".gitignore", tmpl.GitIgnore},
		},
	}

	if err := create(c); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Verify files exist
	for _, f := range c.Files {
		path := filepath.Join(svcDir, f.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f.Path)
		}
	}

	// Verify main.go content
	mainContent, err := os.ReadFile(filepath.Join(svcDir, "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	content := string(mainContent)
	if !strings.Contains(content, `micro.New("mysvc"`) {
		t.Error("main.go should contain service name")
	}
	if !strings.Contains(content, "mcp.WithMCP") {
		t.Error("main.go should contain MCP integration")
	}
	if !strings.Contains(content, "type Mysvc struct") {
		t.Error("main.go should contain handler struct")
	}
}

func TestCreateSimpleServiceNoMCP(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "mysvc")

	c := config{
		Alias:  "mysvc",
		Dir:    svcDir,
		GoDir:  svcDir,
		GoPath: dir,
		Files: []file{
			{"main.go", tmpl.SimpleMain},
			{"Makefile", tmpl.SimpleMakefile},
			{".gitignore", tmpl.GitIgnore},
		},
	}

	if err := create(c); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mainContent, err := os.ReadFile(filepath.Join(svcDir, "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	content := string(mainContent)
	if !strings.Contains(content, `micro.New("mysvc"`) {
		t.Error("main.go should contain service name")
	}
	if strings.Contains(content, "mcp.WithMCP") {
		t.Error("main.go should NOT contain MCP when noMCP is set")
	}
}

func TestCreateProtoService(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "mysvc")

	c := config{
		Alias:  "mysvc",
		Dir:    svcDir,
		GoDir:  svcDir,
		GoPath: dir,
		Files: []file{
			{"main.go", tmpl.MainSRV},
			{"handler/mysvc.go", tmpl.HandlerSRV},
			{"proto/mysvc.proto", tmpl.ProtoSRV},
			{"Makefile", tmpl.Makefile},
			{".gitignore", tmpl.GitIgnore},
		},
	}

	if err := create(c); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	for _, f := range c.Files {
		path := filepath.Join(svcDir, f.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f.Path)
		}
	}
}

func TestCreateFailsIfDirExists(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "mysvc")
	os.MkdirAll(svcDir, 0755)

	c := config{
		Alias: "mysvc",
		Dir:   svcDir,
		Files: []file{{"main.go", tmpl.SimpleMain}},
	}

	err := create(c)
	if err == nil {
		t.Fatal("expected error when directory already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}
