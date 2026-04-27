// Package doctor provides the 'micro doctor' diagnostic command
package doctor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/registry"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "doctor",
		Usage: "Diagnose common issues with your go-micro setup",
		Description: `Run diagnostic checks on your go-micro environment.

Checks Go installation, dependencies, registry connectivity,
port availability, and common configuration issues.

Examples:
  micro doctor
  micro doctor --verbose`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show detailed output for each check",
			},
		},
		Action: doctorAction,
	})
}

type checkResult struct {
	name    string
	ok      bool
	message string
	detail  string
}

func doctorAction(c *cli.Context) error {
	verbose := c.Bool("verbose")

	fmt.Println("micro doctor")
	fmt.Println("============")
	fmt.Println()

	checks := []checkResult{
		checkGo(verbose),
		checkGoModule(verbose),
		checkProtoc(verbose),
		checkRegistry(verbose),
		checkCommonPorts(verbose),
		checkNATS(verbose),
		checkMicroConfig(verbose),
	}

	passed := 0
	failed := 0
	warned := 0

	for _, check := range checks {
		if check.ok {
			fmt.Printf("  [OK] %s\n", check.message)
			passed++
		} else if strings.HasPrefix(check.message, "[WARN]") {
			fmt.Printf("  %s\n", check.message)
			warned++
		} else {
			fmt.Printf("  [FAIL] %s\n", check.message)
			failed++
		}
		if verbose && check.detail != "" {
			for _, line := range strings.Split(check.detail, "\n") {
				fmt.Printf("         %s\n", line)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Results: %d passed, %d warnings, %d failed\n", passed, warned, failed)

	if failed > 0 {
		fmt.Println()
		fmt.Println("Run 'micro doctor --verbose' for details on failures.")
		return fmt.Errorf("%d check(s) failed", failed)
	}

	fmt.Println()
	fmt.Println("Everything looks good!")
	return nil
}

func checkGo(verbose bool) checkResult {
	out, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return checkResult{
			name:    "go",
			ok:      false,
			message: "Go not found in PATH",
			detail:  "Install Go from https://go.dev/dl/",
		}
	}
	version := strings.TrimSpace(string(out))
	return checkResult{
		name:    "go",
		ok:      true,
		message: fmt.Sprintf("Go installed (%s, %s/%s)", version, runtime.GOOS, runtime.GOARCH),
	}
}

func checkGoModule(verbose bool) checkResult {
	// Check if we're in a Go module
	if _, err := os.Stat("go.mod"); err != nil {
		return checkResult{
			name:    "module",
			ok:      false,
			message: "[WARN] No go.mod in current directory",
			detail:  "Run 'go mod init <module>' or 'micro new <name>' to create a project",
		}
	}

	data, err := os.ReadFile("go.mod")
	if err != nil {
		return checkResult{name: "module", ok: false, message: "Cannot read go.mod"}
	}

	hasMicro := strings.Contains(string(data), "go-micro.dev/v5")
	if !hasMicro {
		return checkResult{
			name:    "module",
			ok:      false,
			message: "[WARN] go.mod does not reference go-micro.dev/v5",
			detail:  "Run 'go get go-micro.dev/v5' to add it",
		}
	}

	return checkResult{
		name:    "module",
		ok:      true,
		message: "Go module with go-micro dependency found",
	}
}

func checkProtoc(verbose bool) checkResult {
	_, err := exec.LookPath("protoc")
	if err != nil {
		return checkResult{
			name:    "protoc",
			ok:      false,
			message: "[WARN] protoc not found (optional, needed for --proto services)",
			detail:  "Install from https://grpc.io/docs/protoc-installation/\nOnly needed if using 'micro new --proto'",
		}
	}
	return checkResult{name: "protoc", ok: true, message: "protoc installed"}
}

func checkRegistry(verbose bool) checkResult {
	start := time.Now()
	services, err := registry.ListServices()
	elapsed := time.Since(start)

	if err != nil {
		return checkResult{
			name:    "registry",
			ok:      false,
			message: fmt.Sprintf("Registry unavailable: %v", err),
			detail:  "Default registry is mDNS (works without setup).\nFor Consul: docker run -p 8500:8500 consul:latest agent -dev",
		}
	}

	return checkResult{
		name:    "registry",
		ok:      true,
		message: fmt.Sprintf("Registry reachable (%d services, %s)", len(services), elapsed.Round(time.Millisecond)),
	}
}

func checkCommonPorts(verbose bool) checkResult {
	ports := []string{"8080", "9001", "9002"}
	inUse := []string{}

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", "localhost:"+port, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			inUse = append(inUse, port)
		}
	}

	if len(inUse) > 0 {
		return checkResult{
			name:    "ports",
			ok:      false,
			message: fmt.Sprintf("[WARN] Ports in use: %s", strings.Join(inUse, ", ")),
			detail:  "These ports are commonly used by go-micro services.\nUse micro.Address(\":PORT\") to pick a different port.",
		}
	}

	return checkResult{
		name:    "ports",
		ok:      true,
		message: fmt.Sprintf("Common ports available (%s)", strings.Join(ports, ", ")),
	}
}

func checkNATS(verbose bool) checkResult {
	conn, err := net.DialTimeout("tcp", "localhost:4222", 500*time.Millisecond)
	if err != nil {
		return checkResult{
			name:    "nats",
			ok:      false,
			message: "[WARN] NATS not reachable on localhost:4222 (optional)",
			detail:  "NATS is optional but needed for broker/nats and events/natsjs.\nStart with: docker run -p 4222:4222 nats:latest",
		}
	}
	conn.Close()
	return checkResult{
		name:    "nats",
		ok:      true,
		message: "NATS reachable on localhost:4222",
	}
}

func checkMicroConfig(verbose bool) checkResult {
	// Check for micro.mu or micro.json
	configs := []string{"micro.mu", "micro.json"}
	for _, name := range configs {
		if _, err := os.Stat(name); err == nil {
			absPath, _ := filepath.Abs(name)
			return checkResult{
				name:    "config",
				ok:      true,
				message: fmt.Sprintf("Project config found: %s", absPath),
			}
		}
	}

	return checkResult{
		name:    "config",
		ok:      false,
		message: "[WARN] No micro.mu or micro.json found (optional)",
		detail:  "Project config is optional. Needed for 'micro run' with multiple services.",
	}
}
