package generate

import (
	"bufio"
	"os"
	"strings"

	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/asim/go-micro/cmd/gomu/generate"
	tmpl "github.com/asim/go-micro/cmd/gomu/generate/template"
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

	files := []generate.File{
		{".dockerignore", tmpl.DockerIgnore},
		{"go.mod", tmpl.Module},
		{"plugins.go", tmpl.Plugins},
		{"resources/clusterrole.yaml", tmpl.KubernetesClusterRole},
		{"resources/configmap.yaml", tmpl.KubernetesEnv},
		{"resources/deployment.yaml", tmpl.KubernetesDeployment},
		{"resources/rolebinding.yaml", tmpl.KubernetesRoleBinding},
		{"skaffold.yaml", tmpl.SkaffoldCFG},
	}

	c := generate.Config{
		Service:  service,
		Dir:      ".",
		Vendor:   vendor,
		Comments: []string{"skaffold project template files generated"},
		Client:   strings.HasSuffix(service, "-client"),
		Jaeger:   false,
		Skaffold: true,
	}

	generate.Create(files, c)

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
