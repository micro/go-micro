package new

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/asim/go-micro/cmd/gomu/generate"
	tmpl "github.com/asim/go-micro/cmd/gomu/generate/template"
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

// NewCommand returns a new new cli command.
func init() {
	cmd.Register(&cli.Command{
		Name:  "new",
		Usage: "Create a project template",
		Subcommands: []*cli.Command{
			{
				Name:   "client",
				Usage:  "Create a client template, e.g. " + cmd.App().Name + " new client [github.com/auditemarlow/]helloworld",
				Action: Client,
				Flags:  flags,
			},
			{
				Name:   "function",
				Usage:  "Create a function template, e.g. " + cmd.App().Name + " new function [github.com/auditemarlow/]helloworld",
				Action: Function,
				Flags:  flags,
			},
			{
				Name:   "service",
				Usage:  "Create a service template, e.g. " + cmd.App().Name + " new service [github.com/auditemarlow/]helloworld",
				Action: Service,
				Flags:  flags,
			},
		},
	})
}

func Client(ctx *cli.Context) error {
	return createProject(ctx, "client")
}

// Function creates a new function project template. Exits on error.
func Function(ctx *cli.Context) error {
	return createProject(ctx, "function")
}

// Service creates a new service project template. Exits on error.
func Service(ctx *cli.Context) error {
	return createProject(ctx, "service")
}

func createProject(ctx *cli.Context, pt string) error {
	arg := ctx.Args().First()
	if len(arg) == 0 {
		return cli.ShowSubcommandHelp(ctx)
	}

	name, vendor := getNameAndVendor(arg)

	dir := name
	if pt == "client" {
		dir += "-client"
	}

	if path.IsAbs(dir) {
		fmt.Println("must provide a relative path as service name")
		return nil
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", dir)
	}

	fmt.Printf("creating %s %s\n", pt, name)

	files := []generate.File{
		{".dockerignore", tmpl.DockerIgnore},
		{".gitignore", tmpl.GitIgnore},
		{"Dockerfile", tmpl.Dockerfile},
		{"Makefile", tmpl.Makefile},
		{"go.mod", tmpl.Module},
	}

	switch pt {
	case "client":
		files = append(files, []generate.File{
			{"main.go", tmpl.MainCLT},
		}...)
	case "function":
		files = append(files, []generate.File{
			{"handler/" + name + ".go", tmpl.HandlerFNC},
			{"main.go", tmpl.MainFNC},
			{"proto/" + name + ".proto", tmpl.ProtoFNC},
		}...)
	case "service":
		files = append(files, []generate.File{
			{"handler/" + name + ".go", tmpl.HandlerSRV},
			{"main.go", tmpl.MainSRV},
			{"proto/" + name + ".proto", tmpl.ProtoSRV},
		}...)
	default:
		return fmt.Errorf("%s project type not supported", pt)
	}

	if ctx.Bool("skaffold") {
		files = append(files, []generate.File{
			{"plugins.go", tmpl.Plugins},
			{"resources/clusterrole.yaml", tmpl.KubernetesClusterRole},
			{"resources/configmap.yaml", tmpl.KubernetesEnv},
			{"resources/deployment.yaml", tmpl.KubernetesDeployment},
			{"resources/rolebinding.yaml", tmpl.KubernetesRoleBinding},
			{"skaffold.yaml", tmpl.SkaffoldCFG},
		}...)
	}

	c := generate.Config{
		Service:  name,
		Vendor:   vendor,
		Dir:      dir,
		Client:   pt == "client",
		Jaeger:   ctx.Bool("jaeger"),
		Skaffold: ctx.Bool("skaffold"),
	}

	if pt == "client" {
		c.Comments = clientComments(name, dir)
	} else {
		c.Comments = protoComments(name, dir)
	}

	return create(files, c)
}

func clientComments(name, dir string) []string {
	return []string{
		"cd " + dir,
		"make tidy\n",
	}
}

func protoComments(name, dir string) []string {
	return []string{
		"\ndownload protoc zip packages (protoc-$VERSION-$PLATFORM.zip) and install:\n",
		"visit https://github.com/protocolbuffers/protobuf/releases/latest",
		"\ndownload protobuf for go-micro:\n",
		"go get -u google.golang.org/protobuf/proto",
		"go install github.com/golang/protobuf/protoc-gen-go@latest",
		"go install github.com/asim/go-micro/cmd/protoc-gen-micro/v3@latest",
		"\ncompile the proto file " + name + ".proto:\n",
		"cd " + dir,
		"make proto tidy\n",
	}
}

func create(files []generate.File, c generate.Config) error {
	generate.Create(files, c)

	for _, comment := range c.Comments {
		fmt.Println(comment)
	}

	return nil
}

func getNameAndVendor(s string) (string, string) {
	var n string
	var v string

	if i := strings.LastIndex(s, "/"); i == -1 {
		n = s
		v = ""
	} else {
		n = s[i+1:]
		v = s[:i+1]
	}

	return n, v
}
