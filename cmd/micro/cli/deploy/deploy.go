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
	if sshTarget == "" {
		return fmt.Errorf("specify target with --ssh user@host")
	}

	return deploySSH(c, sshTarget)
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

	// Load config
	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	remotePath := c.String("path")
	if remotePath == "" {
		remotePath = "~/micro"
	}

	// Check if we have pre-built binaries
	binDir := filepath.Join(absDir, "bin")
	hasBinaries := false
	if _, err := os.Stat(binDir); err == nil {
		hasBinaries = true
	}

	fmt.Printf("Deploying to %s...\n", target)

	// Create remote directory
	fmt.Println("Creating remote directory...")
	if err := runSSH(target, fmt.Sprintf("mkdir -p %s/bin", remotePath)); err != nil {
		return err
	}

	if hasBinaries && !c.Bool("build") {
		// Deploy pre-built binaries
		fmt.Println("Copying binaries...")
		if err := copyBinaries(target, binDir, remotePath); err != nil {
			return err
		}
	} else {
		// Sync source and build on remote
		fmt.Println("Syncing source code...")
		if err := syncSource(target, absDir, remotePath); err != nil {
			return err
		}

		fmt.Println("Building on remote...")
		if err := buildOnRemote(target, remotePath, cfg); err != nil {
			return err
		}
	}

	// Stop and start services
	fmt.Println("Restarting services...")
	if err := restartServices(target, remotePath, cfg); err != nil {
		return err
	}

	fmt.Printf("\n✓ Deployed to %s\n", target)
	fmt.Printf("\nView logs: ssh %s 'tail -f %s/logs/*.log'\n", target, remotePath)

	return nil
}

func copyBinaries(target, binDir, remotePath string) error {
	// Use scp to copy binaries
	scpArgs := []string{
		"-r",
		binDir + "/",
		fmt.Sprintf("%s:%s/bin/", target, remotePath),
	}
	scpCmd := exec.Command("scp", scpArgs...)
	scpCmd.Stdout = os.Stdout
	scpCmd.Stderr = os.Stderr
	return scpCmd.Run()
}

func syncSource(target, absDir, remotePath string) error {
	rsyncArgs := []string{
		"-avz", "--delete",
		"--exclude", ".git",
		"--exclude", "bin",
		"--exclude", "node_modules",
		"--exclude", "vendor",
		absDir + "/",
		fmt.Sprintf("%s:%s/src/", target, remotePath),
	}
	rsyncCmd := exec.Command("rsync", rsyncArgs...)
	rsyncCmd.Stdout = os.Stdout
	rsyncCmd.Stderr = os.Stderr
	return rsyncCmd.Run()
}

func buildOnRemote(target, remotePath string, cfg *config.Config) error {
	if cfg != nil && len(cfg.Services) > 0 {
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return err
		}

		for _, svc := range sorted {
			srcPath := filepath.Join(remotePath, "src", svc.Path)
			binPath := filepath.Join(remotePath, "bin", svc.Name)

			buildCmd := fmt.Sprintf("cd %s && go build -o %s .", srcPath, binPath)
			fmt.Printf("  Building %s...\n", svc.Name)
			if err := runSSH(target, buildCmd); err != nil {
				return fmt.Errorf("failed to build %s: %w", svc.Name, err)
			}
		}
	} else {
		// Single service
		srcPath := filepath.Join(remotePath, "src")
		binPath := filepath.Join(remotePath, "bin", "service")

		buildCmd := fmt.Sprintf("cd %s && go build -o %s .", srcPath, binPath)
		if err := runSSH(target, buildCmd); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}

	return nil
}

func restartServices(target, remotePath string, cfg *config.Config) error {
	// Create logs directory
	runSSH(target, fmt.Sprintf("mkdir -p %s/logs", remotePath))

	if cfg != nil && len(cfg.Services) > 0 {
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return err
		}

		for _, svc := range sorted {
			binPath := filepath.Join(remotePath, "bin", svc.Name)
			logPath := filepath.Join(remotePath, "logs", svc.Name+".log")

			// Stop existing
			stopCmd := fmt.Sprintf("pkill -f '%s' 2>/dev/null || true", binPath)
			runSSH(target, stopCmd)

			// Start new
			startCmd := fmt.Sprintf("nohup %s >> %s 2>&1 &", binPath, logPath)
			if err := runSSH(target, startCmd); err != nil {
				return fmt.Errorf("failed to start %s: %w", svc.Name, err)
			}

			fmt.Printf("  ✓ %s\n", svc.Name)
		}
	} else {
		binPath := filepath.Join(remotePath, "bin", "service")
		logPath := filepath.Join(remotePath, "logs", "service.log")

		runSSH(target, fmt.Sprintf("pkill -f '%s' 2>/dev/null || true", binPath))

		startCmd := fmt.Sprintf("nohup %s >> %s 2>&1 &", binPath, logPath)
		if err := runSSH(target, startCmd); err != nil {
			return fmt.Errorf("start failed: %w", err)
		}

		fmt.Println("  ✓ service")
	}

	return nil
}

func runSSH(host, command string) error {
	// Expand ~ on remote
	command = strings.Replace(command, "~/", "$HOME/", -1)
	cmd := exec.Command("ssh", host, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "deploy",
		Usage: "Deploy services via SSH",
		Description: `Deploy copies binaries or source to a remote host and starts services.

If ./bin/ exists (from 'micro build'), copies binaries directly.
Otherwise, syncs source and builds on the remote host.

Examples:
  micro build --os linux          # Build Linux binaries locally
  micro deploy --ssh user@host    # Copy binaries and restart

  micro deploy --ssh user@host    # Sync source, build on remote, restart
  micro deploy --ssh user@host --build  # Force rebuild on remote`,
		Action: Deploy,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "ssh",
				Usage:    "Deploy to user@host via SSH",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Remote path (default: ~/micro)",
				Value: "~/micro",
			},
			&cli.BoolFlag{
				Name:  "build",
				Usage: "Force rebuild on remote (ignore local binaries)",
			},
		},
	})
}
