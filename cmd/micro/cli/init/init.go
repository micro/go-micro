// Package initcmd provides the micro init command for server setup
package initcmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
)

const systemdTemplate = `[Unit]
Description=Micro service: %%i
After=network.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
ExecStart=%s/bin/%%i
Restart=on-failure
RestartSec=5
EnvironmentFile=-%s/config/%%i.env

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=micro-%%i

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s/data

[Install]
WantedBy=multi-user.target
`

// Init initializes a server to receive micro deployments
func Init(c *cli.Context) error {
	if !c.Bool("server") {
		return fmt.Errorf("usage: micro init --server\n\nInitialize this machine to receive micro deployments")
	}

	// Check if we're on Linux
	if runtime.GOOS != "linux" {
		return fmt.Errorf("micro init --server is only supported on Linux")
	}

	// Check for remote init
	remoteHost := c.String("remote")
	if remoteHost != "" {
		return initRemote(c, remoteHost)
	}

	basePath := c.String("path")
	userName := c.String("user")

	fmt.Println("Initializing micro server...")
	fmt.Println()

	// Check if running as root (needed for systemd and creating users)
	if os.Geteuid() != 0 {
		return fmt.Errorf(`micro init --server requires root privileges.

Run with sudo:
  sudo micro init --server`)
	}

	// Create user if needed
	if userName == "micro" {
		if err := createMicroUser(); err != nil {
			return err
		}
	}

	// Create directories
	fmt.Println("Creating directories:")
	dirs := []string{
		filepath.Join(basePath, "bin"),
		filepath.Join(basePath, "data"),
		filepath.Join(basePath, "config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
		fmt.Printf("  ✓ %s\n", dir)
	}

	// Set ownership
	if userName != "root" {
		u, err := user.Lookup(userName)
		if err != nil {
			return fmt.Errorf("user %s not found: %w", userName, err)
		}

		// chown -R user:user /opt/micro
		chownCmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", u.Username, u.Username), basePath)
		if err := chownCmd.Run(); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	fmt.Println()

	// Create systemd template
	fmt.Println("Creating systemd template:")
	unitContent := fmt.Sprintf(systemdTemplate, userName, userName, basePath, basePath, basePath, basePath)
	unitPath := "/etc/systemd/system/micro@.service"

	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	fmt.Printf("  ✓ %s\n", unitPath)

	// Reload systemd
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	fmt.Println("  ✓ systemd daemon-reload")

	// Write marker file so deploy can detect initialization
	markerPath := filepath.Join(basePath, ".micro-initialized")
	if err := os.WriteFile(markerPath, []byte("1\n"), 0644); err != nil {
		return fmt.Errorf("failed to write marker: %w", err)
	}

	fmt.Println()
	fmt.Println("Server ready!")
	fmt.Println()
	fmt.Println("  Deploy from your machine:")
	fmt.Printf("    micro deploy user@%s\n", getHostname())
	fmt.Println()
	fmt.Println("  Manage services:")
	fmt.Println("    sudo systemctl status micro@myservice")
	fmt.Println("    sudo journalctl -u micro@myservice -f")
	fmt.Println()

	return nil
}

func createMicroUser() error {
	// Check if user exists
	if _, err := user.Lookup("micro"); err == nil {
		return nil // user already exists
	}

	fmt.Println("Creating micro user:")
	createCmd := exec.Command("useradd", "--system", "--no-create-home", "--shell", "/bin/false", "micro")
	if err := createCmd.Run(); err != nil {
		// Check if it's just because user already exists
		if _, lookupErr := user.Lookup("micro"); lookupErr == nil {
			return nil
		}
		return fmt.Errorf("failed to create micro user: %w", err)
	}
	fmt.Println("  ✓ Created user 'micro'")
	return nil
}

func initRemote(c *cli.Context, host string) error {
	fmt.Printf("Initializing micro on %s...\n\n", host)

	// Check SSH connectivity first
	if err := checkSSH(host); err != nil {
		return err
	}

	basePath := c.String("path")
	userName := c.String("user")

	// Run micro init --server on remote
	initCmd := fmt.Sprintf("sudo micro init --server --path %s --user %s", basePath, userName)

	sshCmd := exec.Command("ssh", host, initCmd)
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Run(); err != nil {
		return fmt.Errorf("remote init failed: %w", err)
	}

	return nil
}

func checkSSH(host string) error {
	// Quick SSH test
	testCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes", host, "echo ok")
	output, err := testCmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf(`✗ Cannot connect to %s

  SSH connection failed. Check that:
  • The server is reachable: ping %s
  • SSH is configured: ssh %s
  • Your key is added: ssh-add -l

  Common fixes:
  • Add SSH key: ssh-copy-id %s
  • Check hostname in ~/.ssh/config

  Error: %s`, host, host, host, host, strings.TrimSpace(string(output)))
	}

	return nil
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "this-server"
	}
	return name
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "init",
		Usage: "Initialize micro for development or server deployment",
		Description: `Initialize micro on a server to receive deployments.

Server setup:
  sudo micro init --server

This creates:
  • /opt/micro/bin/     - service binaries
  • /opt/micro/data/    - persistent data
  • /opt/micro/config/  - environment files
  • systemd template for managing services

Remote setup:
  micro init --server --remote user@host

After init, deploy with:
  micro deploy user@host`,
		Action: Init,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "server",
				Usage: "Initialize as a deployment server",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Base path for micro (default: /opt/micro)",
				Value: "/opt/micro",
			},
			&cli.StringFlag{
				Name:  "user",
				Usage: "User to run services as (default: micro)",
				Value: "micro",
			},
			&cli.StringFlag{
				Name:  "remote",
				Usage: "Initialize a remote server via SSH",
			},
		},
	})
}
