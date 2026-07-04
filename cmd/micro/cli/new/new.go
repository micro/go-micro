// Package new generates micro service templates
package new

import (
	"bufio"
	"context"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"go-micro.dev/v6/cmd/micro/cli/generate"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/xlab/treeprint"
	tmpl "go-micro.dev/v6/cmd/micro/cli/new/template"
)

type config struct {
	// foo
	Alias string
	// github.com/micro/foo
	Dir string
	// $GOPATH/src/github.com/micro/foo
	GoDir string
	// $GOPATH
	GoPath string
	// UseGoPath
	UseGoPath bool
	// MicroVersion is the go-micro version to require in go.mod
	MicroVersion string
	// Files
	Files []file
	// Comments
	Comments []string
}

// microVersion returns the go-micro version this CLI was built from, so a
// generated service requires the same framework version the user is running.
// Falls back to "latest" for local/dev builds (resolved by 'go mod tidy').
func microVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "latest"
	}
	isRelease := func(v string) bool {
		return strings.HasPrefix(v, "v") && !strings.Contains(v, "devel")
	}
	// cmd/micro is part of the go-micro.dev/v6 module, so for an installed
	// binary the main module version is the framework version.
	if bi.Main.Path == "go-micro.dev/v6" && isRelease(bi.Main.Version) {
		return bi.Main.Version
	}
	for _, dep := range bi.Deps {
		if dep.Path == "go-micro.dev/v6" && isRelease(dep.Version) {
			return dep.Version
		}
	}
	return "latest"
}

type file struct {
	Path string
	Tmpl string
}

func write(c config, file, tmpl string) error {
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

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New("f").Funcs(fn).Parse(tmpl)
	if err != nil {
		return err
	}

	return t.Execute(f, c)
}

func create(c config) error {
	// check if dir exists
	if _, err := os.Stat(c.Dir); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", c.Dir)
	}

	fmt.Println()
	fmt.Println("  \033[1mmicro new\033[0m")
	fmt.Println()
	fmt.Printf("  Creating \033[36m%s\033[0m\n\n", c.Alias)

	t := treeprint.New()

	// write the files
	for _, file := range c.Files {
		f := filepath.Join(c.Dir, file.Path)
		dir := filepath.Dir(f)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		addFileToTree(t, file.Path)
		if err := write(c, f, file.Tmpl); err != nil {
			return err
		}
	}

	// print tree
	fmt.Println(t.String())

	for _, comment := range c.Comments {
		fmt.Println(comment)
	}

	// just wait
	<-time.After(time.Millisecond * 250)

	return nil
}

func addFileToTree(root treeprint.Tree, file string) {
	split := strings.Split(file, "/")
	curr := root
	for i := 0; i < len(split)-1; i++ {
		n := curr.FindByValue(split[i])
		if n != nil {
			curr = n
		} else {
			curr = curr.AddBranch(split[i])
		}
	}
	if curr.FindByValue(split[len(split)-1]) == nil {
		curr.AddNode(split[len(split)-1])
	}
}

func Run(ctx *cli.Context) error {
	// Handle --prompt: design services with AI, then generate each one
	if prompt := ctx.String("prompt"); prompt != "" {
		return runPrompt(ctx, prompt)
	}

	dir := ctx.Args().First()
	if len(dir) == 0 {
		fmt.Println("specify service name")
		return nil
	}

	// check if the path is absolute, we don't want this
	// we want to a relative path so we can install in GOPATH
	if path.IsAbs(dir) {
		fmt.Println("require relative path as service will be installed in GOPATH")
		return nil
	}

	var goPath string
	var goDir string

	goPath = build.Default.GOPATH

	// don't know GOPATH, runaway....
	if len(goPath) == 0 {
		fmt.Println("unknown GOPATH")
		return nil
	}

	// attempt to split path if not windows
	if runtime.GOOS == "windows" {
		goPath = strings.Split(goPath, ";")[0]
	} else {
		goPath = strings.Split(goPath, ":")[0]
	}
	goDir = filepath.Join(goPath, "src", path.Clean(dir))

	noMCP := ctx.Bool("no-mcp")
	templateName := ctx.String("template")

	// The default template is protoless: handlers are registered by
	// reflection, so the service builds and runs with no protoc toolchain.
	// --proto opts into Protocol Buffers; the named templates (crud, pubsub,
	// api) are proto-based and imply it.
	useProto := ctx.Bool("proto") || (templateName != "" && templateName != "default")

	c := config{
		Alias:        dir,
		Comments:     nil,
		Dir:          dir,
		GoDir:        goDir,
		GoPath:       goPath,
		UseGoPath:    false,
		MicroVersion: microVersion(),
	}

	if useProto {
		mainTmpl, handlerTmpl, protoTmpl := selectTemplates(templateName, noMCP)
		c.Files = []file{
			{"main.go", mainTmpl},
			{"handler/" + dir + ".go", handlerTmpl},
			{"proto/" + dir + ".proto", protoTmpl},
			{"Makefile", tmpl.Makefile},
			{"README.md", tmpl.Readme},
			{".gitignore", tmpl.GitIgnore},
		}
	} else {
		mainTmpl := tmpl.MainNoProto
		if noMCP {
			mainTmpl = tmpl.MainNoProtoNoMCP
		}
		c.Files = []file{
			{"main.go", mainTmpl},
			{"handler/" + dir + ".go", tmpl.HandlerNoProto},
			{"Makefile", tmpl.MakefileNoProto},
			{"README.md", tmpl.ReadmeNoProto},
			{".gitignore", tmpl.GitIgnore},
		}
	}

	// set gomodule
	if os.Getenv("GO111MODULE") != "off" {
		mod := tmpl.ModuleNoProto
		if useProto {
			mod = tmpl.Module
		}
		c.Files = append(c.Files, file{"go.mod", mod})
	}

	// create the files
	if err := create(c); err != nil {
		return err
	}

	// Resolve dependencies.
	fmt.Println("\nRunning 'go mod tidy'...")
	if err := runInDir(dir, "go mod tidy"); err != nil {
		fmt.Printf("Error running 'go mod tidy': %v\n", err)
	}

	// Generate protobuf code only when the proto workflow is used, and only
	// when the toolchain is present. Otherwise print install instructions
	// rather than failing with a cryptic error.
	if useProto {
		if missing := missingProtoTools(); len(missing) > 0 {
			printProtoInstall(dir, missing)
		} else {
			fmt.Println("Running 'make proto'...")
			if err := runInDir(dir, "make proto"); err != nil {
				fmt.Printf("Error running 'make proto': %v\n", err)
			}
		}
	}

	// Print updated tree including generated files
	fmt.Println("\nProject structure:")
	printTree(dir)

	fmt.Println()
	fmt.Printf("  \033[32m✓\033[0m Service \033[36m%s\033[0m created\n\n", dir)
	printNextSteps(os.Stdout, dir, noMCP)
	return nil
}

func printNextSteps(w io.Writer, dir string, noMCP bool) {
	fmt.Fprintln(w, "  Next steps:")
	fmt.Fprintf(w, "    cd %s\n", dir)
	fmt.Fprintln(w, "    micro agent preflight")
	fmt.Fprintln(w, "    go run .")
	fmt.Fprintln(w, "    micro chat")
	fmt.Fprintln(w, "    micro inspect agent")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  First-agent path:")
	fmt.Fprintln(w, "    micro docs")
	fmt.Fprintln(w, "    https://go-micro.dev/docs/guides/your-first-agent.html")
	fmt.Fprintln(w, "    https://go-micro.dev/docs/guides/zero-to-hero.html")
	if !noMCP {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "    MCP tools   \033[36mhttp://localhost:3001/mcp/tools\033[0m\n")
		fmt.Fprintln(w, "    Claude Code \033[2mmicro mcp serve\033[0m")
	}
	fmt.Fprintln(w)
}

func selectTemplates(name string, noMCP bool) (mainTmpl, handlerTmpl, protoTmpl string) {
	switch name {
	case "crud":
		if noMCP {
			mainTmpl = tmpl.MainSRVNoMCP
		} else {
			mainTmpl = tmpl.MainSRV
		}
		return mainTmpl, tmpl.CrudHandlerSRV, tmpl.CrudProtoSRV
	case "pubsub":
		if noMCP {
			mainTmpl = tmpl.PubsubMainSRVNoMCP
		} else {
			mainTmpl = tmpl.PubsubMainSRV
		}
		return mainTmpl, tmpl.PubsubHandlerSRV, tmpl.PubsubProtoSRV
	case "api":
		if noMCP {
			mainTmpl = tmpl.MainSRVNoMCP
		} else {
			mainTmpl = tmpl.MainSRV
		}
		return mainTmpl, tmpl.ApiHandlerSRV, tmpl.ApiProtoSRV
	default:
		if noMCP {
			mainTmpl = tmpl.MainSRVNoMCP
		} else {
			mainTmpl = tmpl.MainSRV
		}
		return mainTmpl, tmpl.HandlerSRV, tmpl.ProtoSRV
	}
}

// missingProtoTools returns the protobuf tools needed by `make proto` that
// are not on the PATH.
func missingProtoTools() []string {
	var missing []string
	for _, tool := range []string{"protoc", "protoc-gen-go", "protoc-gen-micro"} {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	return missing
}

// printProtoInstall tells the user exactly what to install to generate the
// protobuf code, instead of failing with a cryptic plugin error.
func printProtoInstall(dir string, missing []string) {
	fmt.Println()
	fmt.Printf("  \033[33m!\033[0m This service uses Protocol Buffers, but these tools are missing: %s\n", strings.Join(missing, ", "))
	fmt.Println()
	fmt.Println("  Install them:")
	fmt.Println("    protoc            https://github.com/protocolbuffers/protobuf/releases (or via your package manager)")
	fmt.Println("    protoc-gen-go     go install google.golang.org/protobuf/cmd/protoc-gen-go@latest")
	fmt.Println("    protoc-gen-micro  go install go-micro.dev/v6/cmd/protoc-gen-micro@latest")
	fmt.Println()
	fmt.Printf("  Then generate the code:\n    cd %s && make proto && go run .\n", dir)
	fmt.Println()
}

func runInDir(dir, cmd string) error {
	parts := strings.Fields(cmd)
	c := exec.Command(parts[0], parts[1:]...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func printTree(dir string) {
	t := treeprint.New()
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		parts := strings.Split(rel, string(os.PathSeparator))
		curr := t
		for i := 0; i < len(parts)-1; i++ {
			n := curr.FindByValue(parts[i])
			if n != nil {
				curr = n
			} else {
				curr = curr.AddBranch(parts[i])
			}
		}
		if !info.IsDir() {
			curr.AddNode(parts[len(parts)-1])
		}
		return nil
	}
	_ = filepath.Walk(dir, walk)
	fmt.Println(t.String())
}

func runPrompt(cliCtx *cli.Context, prompt string) error {
	provider := cliCtx.String("provider")
	apiKey := cliCtx.String("api_key")
	if apiKey == "" {
		// Try provider-specific env vars
		for _, env := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY",
			"ATLASCLOUD_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "TOGETHER_API_KEY", "MICRO_AI_API_KEY"} {
			if v := os.Getenv(env); v != "" {
				apiKey = v
				break
			}
		}
	}
	if apiKey == "" {
		return fmt.Errorf("--api_key or a provider API key env var is required for --prompt")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println()
	fmt.Println("  \033[1mmicro new --prompt\033[0m")
	fmt.Println()
	fmt.Printf("  \033[2mDesigning services for:\033[0m %s\n\n", prompt)

	design, err := generate.Design(ctx, provider, apiKey, "", ".", prompt)
	if err != nil {
		return fmt.Errorf("design failed: %w", err)
	}

	fmt.Println("  Services:")
	for _, svc := range design.Services {
		fmt.Printf("    \033[32m●\033[0m \033[36m%s\033[0m — %s\n", svc.Name, svc.Description)
		for _, ep := range svc.Endpoints {
			fmt.Printf("      %s: %s\n", ep.Name, ep.Description)
		}
	}
	fmt.Println()

	if !confirmGenerate() {
		fmt.Println("  Canceled.")
		return nil
	}

	fmt.Println("  Generating code...")
	if err := generate.Generate(ctx, ".", design, provider, apiKey, ""); err != nil {
		return fmt.Errorf("generate failed: %w", err)
	}

	for _, svc := range design.Services {
		fmt.Printf("    \033[32m✓\033[0m %s/\n", svc.Name)
	}
	fmt.Println()

	fmt.Println("  \033[32m✓\033[0m All services generated")
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    micro run                          \033[2m# start all services\033[0m")
	fmt.Println("    micro chat --provider anthropic    \033[2m# talk to them\033[0m")
	fmt.Println()
	return nil
}

func confirmGenerate() bool {
	fmt.Print("  Generate? [Y/n] ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "" || answer == "y" || answer == "yes"
}
