// Package deploy provides the micro deploy command for deploying services
package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/cmd/micro/run/config"
)

// Deploy deploys services to a target
func Deploy(c *cli.Context) error {
	sshTarget := c.String("ssh")
	if sshTarget != "" {
		return deploySSH(c, sshTarget)
	}

	// Default: docker-compose up
	return deployCompose(c)
}

func deployCompose(c *cli.Context) error {
	dir := c.Args().Get(0)
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	composePath := filepath.Join(absDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found. Run 'micro build --compose' first")
	}

	fmt.Println("Deploying with docker-compose...")

	args := []string{"compose", "-f", composePath, "up", "-d"}
	if c.Bool("build") {
		args = append(args, "--build")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = absDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose failed: %w", err)
	}

	fmt.Println("\n✓ Deployed successfully")
	fmt.Println("\nView logs: docker compose logs -f")
	fmt.Println("Stop: docker compose down")

	return nil
}

func deploySSH(c *cli.Context, target string) error {
	dir := c.Args().Get(0)
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load config to get service info
	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	remotePath := c.String("path")
	if remotePath == "" {
		remotePath = "~/micro"
	}

	fmt.Printf("Deploying to %s...\n", target)

	// Parse target: user@host or just host
	var sshHost string
	if strings.Contains(target, "@") {
		sshHost = target
	} else {
		sshHost = target
	}

	// Create remote directory
	fmt.Println("Creating remote directory...")
	if err := runSSH(sshHost, fmt.Sprintf("mkdir -p %s", remotePath)); err != nil {
		return err
	}

	// Sync files using rsync
	fmt.Println("Syncing files...")
	rsyncArgs := []string{
		"-avz", "--delete",
		"--exclude", ".git",
		"--exclude", "node_modules",
		"--exclude", "vendor",
		absDir + "/",
		fmt.Sprintf("%s:%s/", sshHost, remotePath),
	}
	rsyncCmd := exec.Command("rsync", rsyncArgs...)
	rsyncCmd.Stdout = os.Stdout
	rsyncCmd.Stderr = os.Stderr
	if err := rsyncCmd.Run(); err != nil {
		return fmt.Errorf("rsync failed: %w", err)
	}

	// Build and run on remote
	fmt.Println("Building on remote...")

	if cfg != nil && len(cfg.Services) > 0 {
		// Build and run each service
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return err
		}

		for _, svc := range sorted {
			svcPath := filepath.Join(remotePath, svc.Path)
			binPath := filepath.Join(remotePath, "bin", svc.Name)

			// Build
			buildCmd := fmt.Sprintf("cd %s && go build -o %s .", svcPath, binPath)
			if err := runSSH(sshHost, buildCmd); err != nil {
				return fmt.Errorf("failed to build %s: %w", svc.Name, err)
			}

			// Stop existing if running
			stopCmd := fmt.Sprintf("pkill -f '%s' || true", binPath)
			runSSH(sshHost, stopCmd)

			// Start in background
			startCmd := fmt.Sprintf("nohup %s > %s/%s.log 2>&1 &", binPath, remotePath, svc.Name)
			if err := runSSH(sshHost, startCmd); err != nil {
				return fmt.Errorf("failed to start %s: %w", svc.Name, err)
			}

			fmt.Printf("✓ Deployed %s\n", svc.Name)
		}
	} else {
		// Single service
		name := filepath.Base(absDir)
		binPath := filepath.Join(remotePath, "bin", name)

		buildCmd := fmt.Sprintf("cd %s && mkdir -p bin && go build -o %s .", remotePath, binPath)
		if err := runSSH(sshHost, buildCmd); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		stopCmd := fmt.Sprintf("pkill -f '%s' || true", binPath)
		runSSH(sshHost, stopCmd)

		startCmd := fmt.Sprintf("nohup %s > %s/%s.log 2>&1 &", binPath, remotePath, name)
		if err := runSSH(sshHost, startCmd); err != nil {
			return fmt.Errorf("start failed: %w", err)
		}

		fmt.Printf("✓ Deployed %s\n", name)
	}

	fmt.Printf("\n✓ Deployed to %s\n", target)
	fmt.Printf("\nView logs: ssh %s 'tail -f %s/*.log'\n", sshHost, remotePath)

	return nil
}

func runSSH(host, command string) error {
	cmd := exec.Command("ssh", host, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "deploy",
		Usage: "Deploy services to a target",
		Description: `Deploy services using docker-compose or SSH.

Examples:
  micro deploy                   # Deploy with docker-compose
  micro deploy --ssh user@host   # Deploy via SSH
  micro deploy --build           # Rebuild before deploying`,
		Action: Deploy,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ssh",
				Usage: "Deploy via SSH to user@host",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Remote path for SSH deploy (default: ~/micro)",
				Value: "~/micro",
			},
			&cli.BoolFlag{
				Name:  "build",
				Usage: "Rebuild before deploying",
			},
		},
	})
}
