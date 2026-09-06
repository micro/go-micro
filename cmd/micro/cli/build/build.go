// Package build provides the micro build command for building service binaries
package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"

	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/cmd/micro/run/config"
)

// Build builds Go binaries for services
func Build(c *cli.Context) error {
	dir := c.Args().Get(0)
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load config
	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Output directory
	outDir := c.String("output")
	if outDir == "" {
		outDir = filepath.Join(absDir, "bin")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// Target OS/ARCH
	targetOS := c.String("os")
	targetArch := c.String("arch")
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	if cfg != nil && len(cfg.Services) > 0 {
		// Build each service from config
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return err
		}

		for _, svc := range sorted {
			svcDir := filepath.Join(absDir, svc.Path)
			if err := buildService(svc.Name, svcDir, outDir, targetOS, targetArch); err != nil {
				return fmt.Errorf("failed to build %s: %w", svc.Name, err)
			}
		}
	} else {
		// Build single service from current directory
		name := filepath.Base(absDir)
		if err := buildService(name, absDir, outDir, targetOS, targetArch); err != nil {
			return err
		}
	}

	fmt.Printf("\n  \033[32m✓\033[0m Built to \033[36m%s\033[0m\n", outDir)
	return nil
}

func buildService(name, dir, outDir, targetOS, targetArch string) error {
	binName := name
	if targetOS == "windows" {
		binName += ".exe"
	}
	outPath := filepath.Join(outDir, binName)

	fmt.Printf("  Building \033[36m%s (%s/%s)...\n", name, targetOS, targetArch)

	// Build command
	buildCmd := exec.Command("go", "build", "-o", outPath, ".")
	buildCmd.Dir = dir
	buildCmd.Env = append(
		os.Environ(),
		"GOOS="+targetOS,
		"GOARCH="+targetArch,
		"CGO_ENABLED=0",
	)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Printf("  \033[32m✓\033[0m %s\n", outPath)
	return nil
}

// Docker builds container images (optional)
func Docker(c *cli.Context) error {
	dir := c.Args().Get(0)
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tag := c.String("tag")
	if tag == "" {
		tag = "latest"
	}
	registry := c.String("registry")
	push := c.Bool("push")

	if cfg != nil && len(cfg.Services) > 0 {
		for name, svc := range cfg.Services {
			svcDir := filepath.Join(absDir, svc.Path)
			if err := buildDockerImage(name, svcDir, svc.Port, tag, registry, push); err != nil {
				return fmt.Errorf("failed to build %s: %w", name, err)
			}
		}
	} else {
		name := filepath.Base(absDir)
		if err := buildDockerImage(name, absDir, 8080, tag, registry, push); err != nil {
			return err
		}
	}

	return nil
}

const dockerfileTemplate = `FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SVC
RUN CGO_ENABLED=0 go build -o /service ./$SVC

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /service /service
EXPOSE %d
CMD ["/service"]
`

// findModuleRoot walks up from dir until a go.mod is found.
// Services may live inside a parent module (e.g. examples/ in go-micro);
// the build context must be the module root so go.mod/go.sum resolve.
func findModuleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func buildDockerImage(name, dir string, port int, tag, registry string, push bool) error {
	if port == 0 {
		port = 8080
	}

	moduleRoot := findModuleRoot(dir)
	if moduleRoot == "" {
		moduleRoot = dir
	}
	svcPath, err := filepath.Rel(moduleRoot, dir)
	if err != nil {
		return fmt.Errorf("failed to resolve service path: %w", err)
	}

	// Dockerfile must live inside the build context. Use a distinct name when
	// the service is a subdir of the module root, so we don't clobber the
	// module's own Dockerfile.
	dockerfileName := "Dockerfile"
	if svcPath != "." {
		dockerfileName = ".micro.Dockerfile"
	}
	dockerfilePath := filepath.Join(moduleRoot, dockerfileName)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		fmt.Printf("Generating %s for %s...\n", dockerfileName, name)
		dockerfile := fmt.Sprintf(dockerfileTemplate, port)
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dockerfileName, err)
		}
	}

	imageName := name + ":" + tag
	if registry != "" {
		imageName = registry + "/" + imageName
	}

	fmt.Printf("  Building \033[36m%s...\n", imageName)

	buildCmd := exec.Command("docker", "build",
		"-f", dockerfilePath,
		"--build-arg", "SVC="+svcPath,
		"-t", imageName,
		moduleRoot)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	fmt.Printf("  \033[32m✓\033[0m Built %s\n", imageName)

	if push {
		fmt.Printf("Pushing %s...\n", imageName)
		pushCmd := exec.Command("docker", "push", imageName)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("docker push failed: %w", err)
		}
		fmt.Printf("  \033[32m✓\033[0m Pushed %s\n", imageName)
	}

	return nil
}

// Compose generates docker-compose.yml (optional)
func Compose(c *cli.Context) error {
	dir := c.Args().Get(0)
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg == nil || len(cfg.Services) == 0 {
		return fmt.Errorf("no services found in micro.mu or micro.json")
	}

	registry := c.String("registry")
	tag := c.String("tag")
	if tag == "" {
		tag = "latest"
	}

	var sb strings.Builder
	sb.WriteString("# Generated by micro build --compose\n")
	sb.WriteString("\nservices:\n")

	sorted, err := cfg.TopologicalSort()
	if err != nil {
		return err
	}

	for _, svc := range sorted {
		imageName := svc.Name + ":" + tag
		if registry != "" {
			imageName = registry + "/" + imageName
		}

		fmt.Fprintf(&sb, "  %s:\n", svc.Name)
		fmt.Fprintf(&sb, "    image: %s\n", imageName)

		if svc.Port > 0 {
			fmt.Fprintf(&sb, "    ports:\n      - \"%d:%d\"\n", svc.Port, svc.Port)
		}

		if len(svc.Depends) > 0 {
			sb.WriteString("    depends_on:\n")
			for _, dep := range svc.Depends {
				fmt.Fprintf(&sb, "      - %s\n", dep)
			}
		}

		sb.WriteString("    environment:\n      - MICRO_REGISTRY=mdns\n\n")
	}

	// Default gateway: routes to all services via the shared registry.
	sb.WriteString("  gw:\n")
	sb.WriteString("    # image: ghcr.io/micro/go-micro:latest # prebuilt alternative\n")
	sb.WriteString("    build:\n")
	sb.WriteString("      context: .\n")
	sb.WriteString("      dockerfile: internal/docker/Dockerfile\n")
	sb.WriteString("      additional_contexts:\n        repo: .\n")
	sb.WriteString("    ports:\n      - \"8080:8080\"\n")
	sb.WriteString("    depends_on:\n")
	for _, svc := range sorted {
		fmt.Fprintf(&sb, "      - %s\n", svc.Name)
	}
	sb.WriteString("    environment:\n      - MICRO_REGISTRY=mdns\n      - MICRO_ADMIN_USER=admin\n      - MICRO_ADMIN_PASSWORD=micro\n\n")

	output := filepath.Join(absDir, "docker-compose.yml")
	if err := os.WriteFile(output, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	fmt.Printf("  \033[32m✓\033[0m Generated %s\n", output)
	return nil
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "build",
		Usage: "Build Go binaries for services",
		Description: `Build compiles Go binaries for your services.

With a micro.mu config, builds all services. Without, builds the current directory.
Output goes to ./bin/ by default.

Examples:
  micro build                      # Build for current OS/arch
  micro build --os linux           # Cross-compile for Linux
  micro build --os linux --arch arm64  # For ARM64
  micro build --output ./dist      # Custom output directory

Docker (optional):
  micro build --docker             # Build container images
  micro build --docker --push      # Build and push
  micro build --compose            # Generate docker-compose.yml`,
		Action: func(c *cli.Context) error {
			if c.Bool("docker") {
				return Docker(c)
			}
			if c.Bool("compose") {
				return Compose(c)
			}
			return Build(c)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output directory (default: ./bin)",
			},
			&cli.StringFlag{
				Name:  "os",
				Usage: "Target OS (linux, darwin, windows)",
			},
			&cli.StringFlag{
				Name:  "arch",
				Usage: "Target architecture (amd64, arm64)",
			},
			// Docker options (optional)
			&cli.BoolFlag{
				Name:  "docker",
				Usage: "Build Docker container images instead",
			},
			&cli.StringFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Docker image tag (default: latest)",
				Value:   "latest",
			},
			&cli.StringFlag{
				Name:    "registry",
				Aliases: []string{"r"},
				Usage:   "Docker registry (e.g., docker.io/myuser)",
			},
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push Docker images after building",
			},
			&cli.BoolFlag{
				Name:  "compose",
				Usage: "Generate docker-compose.yml",
			},
		},
	})
}
