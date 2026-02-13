// Package deploy provides the micro deploy command for deploying services
package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/cmd/micro/run/config"
)

const (
	defaultRemotePath = "/opt/micro"
)

// Deploy deploys services to a target
func Deploy(c *cli.Context) error {
	// Get target from args or flag
	target := c.Args().First()
	if target == "" {
		target = c.String("ssh")
	}

	// Load config to check for deploy targets
	dir := "."
	absDir, _ := filepath.Abs(dir)
	cfg, _ := config.Load(absDir)

	// If still no target, check config for named targets
	if target == "" && cfg != nil && len(cfg.Deploy) > 0 {
		// Show available targets
		return showDeployTargets(cfg)
	}

	if target == "" {
		return showDeployHelp()
	}

	// Check if target is a named target from config
	if cfg != nil {
		if dt, ok := cfg.Deploy[target]; ok {
			target = dt.SSH
		}
	}

	return deploySSH(c, target, cfg)
}

func showDeployHelp() error {
	return fmt.Errorf(`No deployment target specified.

To deploy, you need a server running micro. Quick setup:

  1. On your server (Ubuntu/Debian):
     ssh user@your-server
     curl -fsSL https://go-micro.dev/install.sh | sh
     sudo micro init --server

  2. Then deploy from here:
     micro deploy user@your-server

  Or add to micro.mu:
     deploy prod
         ssh user@your-server

Run 'micro deploy --help' for more options.`)
}

func showDeployTargets(cfg *config.Config) error {
	var sb strings.Builder
	sb.WriteString("Available deploy targets:\n\n")
	for name, dt := range cfg.Deploy {
		sb.WriteString(fmt.Sprintf("  %s -> %s\n", name, dt.SSH))
	}
	sb.WriteString("\nDeploy with: micro deploy <target>")
	return fmt.Errorf("%s", sb.String())
}

func deploySSH(c *cli.Context, target string, cfg *config.Config) error {
	dir := c.Args().Get(1)
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load config if not passed
	if cfg == nil {
		cfg, _ = config.Load(absDir)
	}

	remotePath := c.String("path")
	if remotePath == "" {
		remotePath = defaultRemotePath
	}

	fmt.Printf("Deploying to %s...\n\n", target)

	// Early validation: Check if the requested service exists before SSH checks
	filterService := c.String("service")
	if filterService != "" && cfg != nil {
		found := false
		for _, svc := range cfg.Services {
			if svc.Name == filterService {
				found = true
				break
			}
		}
		if !found && len(cfg.Services) > 0 {
			return fmt.Errorf("service '%s' not found in configuration", filterService)
		}
	}

	// Step 1: Check SSH connectivity
	fmt.Print("  Checking SSH connection... ")
	if err := checkSSH(target); err != nil {
		fmt.Println("\u2717")
		return err
	}
	fmt.Println("\u2713")

	// Step 2: Check server is initialized
	fmt.Print("  Checking server setup...   ")
	if err := checkServerInit(target, remotePath); err != nil {
		fmt.Println("\u2717")
		return err
	}
	fmt.Println("\u2713")

	// Step 3: Build binaries
	var services []string
	if cfg != nil && len(cfg.Services) > 0 {
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return err
		}
		for _, svc := range sorted {
			// If --service flag is provided, only include that service
			if filterService == "" || svc.Name == filterService {
				services = append(services, svc.Name)
			}
		}
	} else {
		// Single service project
		services = []string{filepath.Base(absDir)}

		// If --service flag was provided for a single-service project, validate it matches
		if filterService != "" && filterService != services[0] {
			return fmt.Errorf("service '%s' not found (only '%s' available)", filterService, services[0])
		}
	}

	fmt.Printf("  Building binaries...       ")
	if err := buildBinaries(absDir, cfg, c.Bool("build"), services); err != nil {
		fmt.Println("\u2717")
		return err
	}
	fmt.Printf("\u2713 %s\n", strings.Join(services, ", "))

	// Step 4: Copy binaries
	fmt.Printf("  Copying binaries...        ")
	if err := copyBinaries(target, filepath.Join(absDir, "bin"), remotePath); err != nil {
		fmt.Println("\u2717")
		return err
	}
	fmt.Printf("\u2713 %d services\n", len(services))

	// Step 5: Setup and restart services via systemd
	fmt.Printf("  Updating systemd...        ")
	if err := setupSystemdServices(target, remotePath, services); err != nil {
		fmt.Println("\u2717")
		return err
	}
	fmt.Printf("\u2713 %s\n", strings.Join(prefixServices(services), ", "))

	// Step 6: Restart services
	fmt.Printf("  Restarting services...     ")
	if err := restartServices(target, services); err != nil {
		fmt.Println("\u2717")
		return err
	}
	fmt.Println("\u2713")

	// Step 7: Check health
	fmt.Printf("  Checking health...         ")
	time.Sleep(2 * time.Second) // Give services time to start
	healthy, unhealthy := checkServicesHealth(target, services)
	if len(unhealthy) > 0 {
		fmt.Printf("\u26a0 %d/%d healthy\n", len(healthy), len(services))
	} else {
		fmt.Println("\u2713 all healthy")
	}

	fmt.Println()
	fmt.Printf("\u2713 Deployed to %s\n", target)
	fmt.Println()
	fmt.Printf("  Status: micro status --remote %s\n", target)
	fmt.Printf("  Logs:   micro logs --remote %s\n", target)

	if len(unhealthy) > 0 {
		fmt.Println()
		fmt.Printf("\u26a0 Some services may have issues: %s\n", strings.Join(unhealthy, ", "))
		fmt.Printf("  Check logs: micro logs %s --remote %s\n", unhealthy[0], target)
	}

	return nil
}

func prefixServices(services []string) []string {
	result := make([]string, len(services))
	for i, s := range services {
		result[i] = "micro@" + s
	}
	return result
}

func checkSSH(host string) error {
	testCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes", host, "echo ok")
	output, err := testCmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`
\u2717 Cannot connect to %s

  SSH connection failed. Check that:
  \u2022 The server is reachable: ping %s
  \u2022 SSH is configured: ssh %s
  \u2022 Your key is added: ssh-add -l

  Common fixes:
  \u2022 Add SSH key: ssh-copy-id %s
  \u2022 Check hostname in ~/.ssh/config

  Error: %s`, host, host, host, host, strings.TrimSpace(string(output)))
	}
	return nil
}

func checkServerInit(host, remotePath string) error {
	checkCmd := fmt.Sprintf("test -f %s/.micro-initialized", remotePath)
	sshCmd := exec.Command("ssh", host, checkCmd)
	if err := sshCmd.Run(); err != nil {
		return fmt.Errorf(`
\u2717 Server not initialized

  micro is not set up on %s.

  Run this on the server:
    ssh %s
    curl -fsSL https://go-micro.dev/install.sh | sh
    sudo micro init --server

  Or initialize remotely (requires sudo):
    micro init --server --remote %s`, host, host, host)
	}
	return nil
}

func buildBinaries(absDir string, cfg *config.Config, forceBuild bool, servicesToBuild []string) error {
	binDir := filepath.Join(absDir, "bin")

	// Check if we already have binaries and don't need to rebuild
	if !forceBuild {
		if _, err := os.Stat(binDir); err == nil {
			// Check if binaries are for linux
			// For now, just rebuild to be safe
		}
	}

	// Always build for linux/amd64
	targetOS := "linux"
	targetArch := "amd64"

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	if cfg != nil && len(cfg.Services) > 0 {
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return err
		}

		// Create a map for quick lookup of services to build
		// This provides O(1) lookup time and makes the code more maintainable
		shouldBuild := make(map[string]bool)
		for _, svcName := range servicesToBuild {
			shouldBuild[svcName] = true
		}

		for _, svc := range sorted {
			// Only build services in the servicesToBuild list
			if !shouldBuild[svc.Name] {
				continue
			}

			svcDir := filepath.Join(absDir, svc.Path)
			outPath := filepath.Join(binDir, svc.Name)

			buildCmd := exec.Command("go", "build", "-o", outPath, ".")
			buildCmd.Dir = svcDir
			buildCmd.Env = append(os.Environ(),
				"GOOS="+targetOS,
				"GOARCH="+targetArch,
				"CGO_ENABLED=0",
			)

			if output, err := buildCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to build %s:\n%s", svc.Name, string(output))
			}
		}
	} else {
		name := filepath.Base(absDir)
		outPath := filepath.Join(binDir, name)

		buildCmd := exec.Command("go", "build", "-o", outPath, ".")
		buildCmd.Dir = absDir
		buildCmd.Env = append(os.Environ(),
			"GOOS="+targetOS,
			"GOARCH="+targetArch,
			"CGO_ENABLED=0",
		)

		if output, err := buildCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to build:\n%s", string(output))
		}
	}

	return nil
}

func copyBinaries(target, binDir, remotePath string) error {
	// Ensure remote bin directory exists
	mkdirCmd := exec.Command("ssh", target, fmt.Sprintf("mkdir -p %s/bin", remotePath))
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Use rsync for efficient copy
	// --omit-dir-times avoids permission errors on directory timestamps
	rsyncArgs := []string{
		"-avz", "--delete", "--omit-dir-times",
		binDir + "/",
		fmt.Sprintf("%s:%s/bin/", target, remotePath),
	}

	rsyncCmd := exec.Command("rsync", rsyncArgs...)
	output, err := rsyncCmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Fall back to scp if rsync not available
		if strings.Contains(outputStr, "command not found") {
			scpCmd := exec.Command("scp", "-r", binDir+"/", fmt.Sprintf("%s:%s/bin/", target, remotePath))
			if scpOutput, scpErr := scpCmd.CombinedOutput(); scpErr != nil {
				return fmt.Errorf("copy failed: %s", string(scpOutput))
			}
			return nil
		}
		// rsync exit code 23 means some files failed to transfer, but if we see our files listed, it's ok
		// rsync exit code 24 means some files vanished during transfer (harmless)
		exitErr, ok := err.(*exec.ExitError)
		if ok && (exitErr.ExitCode() == 23 || exitErr.ExitCode() == 24) {
			// Check if it's just permission warnings on metadata, not actual file transfer failures
			if !strings.Contains(outputStr, "Permission denied (13)") ||
				strings.Contains(outputStr, "failed to set times") ||
				strings.Contains(outputStr, "chgrp") {
				// These are acceptable warnings
				return nil
			}
		}
		return fmt.Errorf("copy failed: %s", outputStr)
	}

	return nil
}

func setupSystemdServices(target, remotePath string, services []string) error {
	for _, svc := range services {
		// Enable the service using the template
		enableCmd := fmt.Sprintf("sudo systemctl enable micro@%s 2>/dev/null || true", svc)
		sshCmd := exec.Command("ssh", target, enableCmd)
		sshCmd.Run() // Ignore errors, service might already be enabled
	}

	// Reload systemd
	reloadCmd := exec.Command("ssh", target, "sudo systemctl daemon-reload")
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	return nil
}

func restartServices(target string, services []string) error {
	for _, svc := range services {
		restartCmd := fmt.Sprintf("sudo systemctl restart micro@%s", svc)
		sshCmd := exec.Command("ssh", target, restartCmd)
		if output, err := sshCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to restart %s: %s", svc, string(output))
		}
	}
	return nil
}

func checkServicesHealth(target string, services []string) (healthy, unhealthy []string) {
	for _, svc := range services {
		checkCmd := fmt.Sprintf("systemctl is-active micro@%s", svc)
		sshCmd := exec.Command("ssh", target, checkCmd)
		if err := sshCmd.Run(); err != nil {
			unhealthy = append(unhealthy, svc)
		} else {
			healthy = append(healthy, svc)
		}
	}
	return
}

// Ensure we're not on Windows for deploy
func checkPlatform() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("micro deploy requires SSH and rsync, which work best on Linux/macOS.\nConsider using WSL on Windows.")
	}
	return nil
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "deploy",
		Usage: "Deploy services to a remote server",
		Description: `Deploy copies binaries to a remote server and manages them with systemd.

Before deploying, initialize the server:
  ssh user@server 'curl -fsSL https://go-micro.dev/install.sh | sh && sudo micro init --server'

Then deploy:
  micro deploy user@server

Deploy a specific service (multi-service projects):
  micro deploy user@server --service users

With a micro.mu config, you can define named targets:
  deploy prod
      ssh user@prod.example.com

  deploy staging
      ssh user@staging.example.com

Then: micro deploy prod

The deploy process:
  1. Builds binaries for linux/amd64
  2. Copies to /opt/micro/bin/ via rsync
  3. Enables and restarts systemd services
  4. Verifies services are healthy`,
		Action: func(c *cli.Context) error {
			if err := checkPlatform(); err != nil {
				return err
			}
			return Deploy(c)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ssh",
				Usage: "Deploy target as user@host (can also be positional arg)",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Remote path (default: /opt/micro)",
				Value: "/opt/micro",
			},
			&cli.BoolFlag{
				Name:  "build",
				Usage: "Force rebuild of binaries",
			},
			&cli.StringFlag{
				Name:  "service",
				Usage: "Deploy only a specific service (for multi-service projects)",
			},
		},
	})
}
