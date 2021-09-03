package new

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/asim/go-micro/cmd/gomu/cmd"
	tmpl "github.com/asim/go-micro/cmd/gomu/cmd/cli/new/template"
	"github.com/urfave/cli/v2"
)

var flags []cli.Flag = []cli.Flag{
	&cli.BoolFlag{
		Name:  "jaeger",
		Usage: "generate jaeger tracer files",
	},
	&cli.BoolFlag{
		Name:  "skaffold",
		Usage: "generate skaffold files",
	},
}

type config struct {
	Alias    string
	Comments []string
	Dir      string
	Jaeger   bool
	Skaffold bool
}

type file struct {
	Path string
	Tmpl string
}

// NewCommand returns a new new cli command.
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "new",
		Usage: "Create a project template",
		Subcommands: []*cli.Command{
			{
				Name:   "function",
				Usage:  "Create a function template, e.g. " + cmd.App().Name + " new function greeter",
				Action: Function,
				Flags:  flags,
			},
			{
				Name:   "service",
				Usage:  "Create a service template, e.g. " + cmd.App().Name + " new service greeter",
				Action: Service,
				Flags:  flags,
			},
		},
	}
}

// Function creates a new function project template. Exits on error.
func Function(ctx *cli.Context) error {
	return createProject(ctx, true)
}

// Service creates a new service project template. Exits on error.
func Service(ctx *cli.Context) error {
	return createProject(ctx, false)
}

func createProject(ctx *cli.Context, fn bool) error {
	name := ctx.Args().First()
	if len(name) == 0 {
		return cli.ShowSubcommandHelp(ctx)
	}

	if path.IsAbs(name) {
		fmt.Println("must provide a relative path as service name")
		return nil
	}

	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", name)
	}

	if fn {
		fmt.Printf("creating function %s\n", name)
	} else {
		fmt.Printf("creating service %s\n", name)
	}

	files := []file{
		{".dockerignore", tmpl.DockerIgnore},
		{".gitignore", tmpl.GitIgnore},
		{"Dockerfile", tmpl.Dockerfile},
		{"Makefile", tmpl.Makefile},
		{"go.mod", tmpl.Module},
	}
	if fn {
		files = append(files, []file{
			{"handler/" + name + ".go", tmpl.HandlerFNC},
			{"main.go", tmpl.MainFNC},
			{"proto/" + name + ".proto", tmpl.ProtoFNC},
		}...)
	} else {
		files = append(files, []file{
			{"handler/" + name + ".go", tmpl.HandlerSRV},
			{"main.go", tmpl.MainSRV},
			{"proto/" + name + ".proto", tmpl.ProtoSRV},
		}...)
	}

	if ctx.Bool("skaffold") {
		files = append(files, []file{
			{"plugins.go", tmpl.Plugins},
			{"resources/clusterrole.yaml", tmpl.KubernetesClusterRole},
			{"resources/configmap.yaml", tmpl.KubernetesEnv},
			{"resources/deployment.yaml", tmpl.KubernetesDeployment},
			{"resources/rolebinding.yaml", tmpl.KubernetesRoleBinding},
			{"skaffold.yaml", tmpl.SkaffoldCFG},
		}...)
	}

	c := config{
		Alias:    name,
		Comments: protoComments(name),
		Dir:      name,
		Jaeger:   ctx.Bool("jaeger"),
		Skaffold: ctx.Bool("skaffold"),
	}

	return create(files, c)
}

func protoComments(alias string) []string {
	return []string{
		"\ndownload protoc zip packages (protoc-$VERSION-$PLATFORM.zip) and install:\n",
		"visit https://github.com/protocolbuffers/protobuf/releases/latest",
		"\ndownload protobuf for go-micro:\n",
		"go get -u google.golang.org/protobuf/proto",
		"go install github.com/golang/protobuf/protoc-gen-go@latest",
		"go install github.com/asim/go-micro/cmd/protoc-gen-micro/v3@latest",
		"\ncompile the proto file " + alias + ".proto:\n",
		"cd " + alias,
		"make proto tidy\n",
	}
}

func create(files []file, c config) error {
	for _, file := range files {
		fp := filepath.Join(c.Alias, file.Path)
		dir := filepath.Dir(fp)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		f, err := os.Create(fp)
		if err != nil {
			return err
		}

		fn := template.FuncMap{
			"dehyphen": func(s string) string {
				return strings.ReplaceAll(s, "-", "")
			},
			"lower": strings.ToLower,
			"title": func(s string) string {
				return strings.ReplaceAll(strings.Title(s), "-", "")
			},
		}
		t, err := template.New(fp).Funcs(fn).Parse(file.Tmpl)
		if err != nil {
			return err
		}

		err = t.Execute(f, c)
		if err != nil {
			return err
		}
	}

	for _, comment := range c.Comments {
		fmt.Println(comment)
	}

	return nil
}
