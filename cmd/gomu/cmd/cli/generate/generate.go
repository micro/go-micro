package generate

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/asim/go-micro/cmd/gomu/file"
	"github.com/asim/go-micro/cmd/gomu/file/generator"
	tmpl "github.com/asim/go-micro/cmd/gomu/file/template"
	"github.com/urfave/cli/v2"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "generate",
		Usage: "Generate project template files after the fact",
		Subcommands: []*cli.Command{
			{
				Name:   "skaffold",
				Usage:  "Generate Skaffold project template files",
				Action: Skaffold,
			},
		},
	})
}

// Skaffold generates Skaffold project template files in the current directory.
// Exits on error.
func Skaffold(ctx *cli.Context) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	service := dir[strings.LastIndex(dir, "/")+1:]

	vendor, err := getServiceVendor(service)
	if err != nil {
		return err
	}

	g := generator.New(
		generator.Service(service),
		generator.Vendor(vendor),
		generator.Directory("."),
		generator.Client(strings.HasSuffix(service, "-client")),
		generator.Skaffold(true),
	)

	files := []file.File{
		{".dockerignore", tmpl.DockerIgnore},
		{"go.mod", tmpl.Module},
		{"plugins.go", tmpl.Plugins},
		{"resources/clusterrole.yaml", tmpl.KubernetesClusterRole},
		{"resources/configmap.yaml", tmpl.KubernetesEnv},
		{"resources/deployment.yaml", tmpl.KubernetesDeployment},
		{"resources/rolebinding.yaml", tmpl.KubernetesRoleBinding},
		{"skaffold.yaml", tmpl.SkaffoldCFG},
	}

	if err := g.Generate(files); err != nil {
		return err
	}

	fmt.Println("skaffold project template files generated")

	return nil
}

func getServiceVendor(s string) (string, error) {
	f, err := os.Open("go.mod")
	if err != nil {
		return "", err
	}
	defer f.Close()

	line := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "module ") {
			line = scanner.Text()
			break

		}
	}
	if line == "" {
		return "", nil
	}

	module := line[strings.LastIndex(line, " ")+1:]
	if module == s {
		return "", nil
	}

	return module[:strings.LastIndex(module, "/")] + "/", nil
}
