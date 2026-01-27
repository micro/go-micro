// Package remote provides remote server operations for micro
package remote

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
)

const defaultRemotePath = "/opt/micro"

// Status shows status of services (local or remote)
func Status(c *cli.Context) error {
	remoteHost := c.String("remote")
	if remoteHost != "" {
		return remoteStatus(remoteHost)
	}
	return localStatus(c)
}

func localStatus(c *cli.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	runDir := filepath.Join(homeDir, "micro", "run")
	files, err := os.ReadDir(runDir)
	if err != nil {
		fmt.Println("No services running locally.")
		fmt.Println("\nStart services with: micro run")
		return nil
	}

	var hasServices bool
	fmt.Printf("%-20s %-10s %-8s %s\n", "SERVICE", "STATUS", "PID", "DIRECTORY")
	fmt.Println(strings.Repeat("-", 70))

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".pid") {
			continue
		}
		hasServices = true
		service := f.Name()[:len(f.Name())-4]
		pidFilePath := filepath.Join(runDir, f.Name())
		pidFile, err := os.Open(pidFilePath)
		if err != nil {
			continue
		}
		var pid int
		var dir string
		scanner := bufio.NewScanner(pidFile)
		if scanner.Scan() {
			fmt.Sscanf(scanner.Text(), "%d", &pid)
		}
		if scanner.Scan() {
			dir = scanner.Text()
		}
		pidFile.Close()

		status := "\u2717 stopped"
		if pid > 0 {
			proc, err := os.FindProcess(pid)
			if err == nil {
				if err := proc.Signal(syscall.Signal(0)); err == nil {
					status = "\u25cf running"
				}
			}
		}
		fmt.Printf("%-20s %-10s %-8d %s\n", service, status, pid, dir)
	}

	if !hasServices {
		fmt.Println("No services running locally.")
		fmt.Println("\nStart services with: micro run")
	}

	return nil
}

func remoteStatus(host string) error {
	// Get list of micro services via systemctl
	listCmd := exec.Command("ssh", host, "systemctl list-units 'micro@*' --no-legend --no-pager 2>/dev/null || true")
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get status from %s: %w", host, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("%s\n", host)
		fmt.Println(strings.Repeat("\u2501", 50))
		fmt.Println("\nNo services deployed.")
		fmt.Println("\nDeploy with: micro deploy " + host)
		return nil
	}

	fmt.Printf("%s\n", host)
	fmt.Println(strings.Repeat("\u2501", 50))
	fmt.Println()

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		unit := parts[0]
		loadState := parts[1]
		activeState := parts[2]
		subState := parts[3]

		// Extract service name from micro@servicename.service
		serviceName := strings.TrimPrefix(unit, "micro@")
		serviceName = strings.TrimSuffix(serviceName, ".service")

		// Get more details
		statusIcon := "\u25cf"
		statusText := subState
		if activeState != "active" || subState != "running" {
			statusIcon = "\u2717"
		}

		_ = loadState // unused but parsed

		fmt.Printf("  %-15s %s %s\n", serviceName, statusIcon, statusText)
	}

	fmt.Println()
	return nil
}

// Logs shows logs for services (local or remote)
func Logs(c *cli.Context) error {
	remoteHost := c.String("remote")
	service := c.Args().First()
	follow := c.Bool("follow") || c.Bool("f")
	lines := c.Int("lines")

	if remoteHost != "" {
		return remoteLogs(remoteHost, service, follow, lines)
	}
	return localLogs(c, service, follow, lines)
}

func localLogs(c *cli.Context, service string, follow bool, lines int) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	logDir := filepath.Join(homeDir, "micro", "logs")

	if service == "" {
		// List available logs
		files, err := os.ReadDir(logDir)
		if err != nil {
			fmt.Println("No logs available.")
			return nil
		}

		fmt.Println("Available logs:")
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".log") {
				name := strings.TrimSuffix(f.Name(), ".log")
				fmt.Printf("  %s\n", name)
			}
		}
		fmt.Println("\nView logs: micro logs <service>")
		return nil
	}

	logPath := filepath.Join(logDir, service+".log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("no logs for service '%s'", service)
	}

	if follow {
		cmd := exec.Command("tail", "-f", logPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if lines == 0 {
		lines = 100
	}
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func remoteLogs(host, service string, follow bool, lines int) error {
	var journalCmd string

	if service == "" {
		// All micro services
		journalCmd = "journalctl -u 'micro@*'"
	} else {
		journalCmd = fmt.Sprintf("journalctl -u 'micro@%s'", service)
	}

	if follow {
		journalCmd += " -f"
	} else {
		if lines == 0 {
			lines = 100
		}
		journalCmd += fmt.Sprintf(" -n %d", lines)
	}

	journalCmd += " --no-pager"

	sshCmd := exec.Command("ssh", host, journalCmd)
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr
	return sshCmd.Run()
}

// Stop stops a running service
func Stop(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("Usage: micro stop <service>")
	}

	service := c.Args().First()
	remoteHost := c.String("remote")

	if remoteHost != "" {
		return remoteStop(remoteHost, service)
	}
	return localStop(service)
}

func localStop(service string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	runDir := filepath.Join(homeDir, "micro", "run")
	pidFilePath := filepath.Join(runDir, service+".pid")

	pidFile, err := os.Open(pidFilePath)
	if err != nil {
		return fmt.Errorf("service '%s' is not running", service)
	}

	var pid int
	scanner := bufio.NewScanner(pidFile)
	if scanner.Scan() {
		fmt.Sscanf(scanner.Text(), "%d", &pid)
	}
	pidFile.Close()

	if pid <= 0 {
		_ = os.Remove(pidFilePath)
		return fmt.Errorf("service '%s' is not running", service)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidFilePath)
		return fmt.Errorf("could not find process for '%s'", service)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = os.Remove(pidFilePath)
		return fmt.Errorf("failed to stop service '%s': %v", service, err)
	}

	_ = os.Remove(pidFilePath)
	fmt.Printf("Stopped %s (pid %d)\n", service, pid)
	return nil
}

func remoteStop(host, service string) error {
	stopCmd := fmt.Sprintf("sudo systemctl stop micro@%s", service)
	sshCmd := exec.Command("ssh", host, stopCmd)
	if output, err := sshCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop %s: %s", service, string(output))
	}
	fmt.Printf("Stopped %s on %s\n", service, host)
	return nil
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "status",
		Usage: "Check status of running services",
		Description: `Show status of running services.

Local status:
  micro status

Remote status:
  micro status --remote user@host`,
		Action: Status,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "remote",
				Usage: "Check status on remote server",
			},
		},
	})

	cmd.Register(&cli.Command{
		Name:  "logs",
		Usage: "Show logs for a service",
		Description: `View service logs.

Local logs:
  micro logs              # list available logs
  micro logs myservice    # show logs for myservice
  micro logs myservice -f # follow logs

Remote logs:
  micro logs --remote user@host
  micro logs myservice --remote user@host -f`,
		Action: Logs,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "remote",
				Usage: "View logs on remote server",
			},
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow log output",
			},
			&cli.IntFlag{
				Name:    "lines",
				Aliases: []string{"n"},
				Usage:   "Number of lines to show (default: 100)",
				Value:   100,
			},
		},
	})

	cmd.Register(&cli.Command{
		Name:  "stop",
		Usage: "Stop a running service",
		Description: `Stop a running service.

Local:
  micro stop myservice

Remote:
  micro stop myservice --remote user@host`,
		Action: Stop,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "remote",
				Usage: "Stop service on remote server",
			},
		},
	})
}
