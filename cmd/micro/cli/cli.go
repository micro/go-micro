package microcli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/registry"

	"go-micro.dev/v5/cmd/micro/cli/new"
	"go-micro.dev/v5/cmd/micro/cli/util"
)

var (
	// version is set by the release action
	// this is the default for local builds
	version = "5.0.0-dev"
)

func lastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}
	return ""
}

func lastLogLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	var last string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		if strings.TrimSpace(scan.Text()) != "" {
			last = scan.Text()
		}
	}
	return last
}

func waitAndCleanup(procs []*exec.Cmd, pidFiles []string) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		for _, proc := range procs {
			if proc.Process != nil {
				_ = proc.Process.Kill()
			}
		}
		for _, pf := range pidFiles {
			_ = os.Remove(pf)
		}
		os.Exit(1)
	}()
	for i, proc := range procs {
		_ = proc.Wait()
		if proc.Process != nil {
			_ = os.Remove(pidFiles[i])
		}
	}
}

func init() {
	cmd.Register([]*cli.Command{
		{
			Name:   "new",
			Usage:  "Create a new service",
			Action: new.Run,
		},
		{
			Name:  "services",
			Usage: "List available services",
			Action: func(ctx *cli.Context) error {
				services, err := registry.ListServices()
				if err != nil {
					return err
				}
				for _, service := range services {
					fmt.Println(service.Name)
				}
				return nil
			},
		},
		{
			Name:  "call",
			Usage: "Call a service",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()

				if args.Len() < 2 {
					return fmt.Errorf("Usage: [service] [endpoint] [request]")
				}

				service := args.Get(0)
				endpoint := args.Get(1)
				request := `{}`

				if args.Len() == 3 {
					request = args.Get(2)
				}

				req := client.NewRequest(service, endpoint, &bytes.Frame{Data: []byte(request)})
				var rsp bytes.Frame
				err := client.Call(context.TODO(), req, &rsp)
				if err != nil {
					return err
				}

				fmt.Print(string(rsp.Data))
				return nil
			},
		},
		{
			Name:  "describe",
			Usage: "Describe a service",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()

				if args.Len() != 1 {
					return fmt.Errorf("Usage: [service]")
				}

				service := args.Get(0)
				services, err := registry.GetService(service)
				if err != nil {
					return err
				}
				if len(services) == 0 {
					return nil
				}
				b, _ := json.MarshalIndent(services[0], "", "    ")
				fmt.Println(string(b))
				return nil
			},
		},
		{
			Name:  "status",
			Usage: "Check status of running services",
			Action: func(ctx *cli.Context) error {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home dir: %w", err)
				}
				runDir := filepath.Join(homeDir, "micro", "run")
				files, err := os.ReadDir(runDir)
				if err != nil {
					return fmt.Errorf("failed to read run dir: %w", err)
				}
				fmt.Printf("%-20s %-8s %-8s %s\n", "SERVICE", "PID", "STATUS", "DIRECTORY")
				for _, f := range files {
					if f.IsDir() || !strings.HasSuffix(f.Name(), ".pid") {
						continue
					}
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
					status := "stopped"
					if pid > 0 {
						proc, err := os.FindProcess(pid)
						if err == nil {
							if err := proc.Signal(syscall.Signal(0)); err == nil {
								status = "running"
							}
						}
					}
					fmt.Printf("%-20s %-8d %-8s %-40s %s\n", service, pid, status, "", dir)
				}
				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "Stop a running service",
			Action: func(ctx *cli.Context) error {
				if ctx.Args().Len() != 1 {
					return fmt.Errorf("Usage: micro stop [service]")
				}
				service := ctx.Args().Get(0)
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home dir: %w", err)
				}
				runDir := filepath.Join(homeDir, "micro", "run")
				pidFilePath := filepath.Join(runDir, service+".pid")
				pidFile, err := os.Open(pidFilePath)
				if err != nil {
					return fmt.Errorf("no pid file for service %s", service)
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
				if pid <= 0 {
					_ = os.Remove(pidFilePath)
					return fmt.Errorf("service %s is not running", service)
				}
				proc, err := os.FindProcess(pid)
				if err != nil {
					_ = os.Remove(pidFilePath)
					return fmt.Errorf("could not find process for %s", service)
				}
				if err := proc.Signal(syscall.SIGTERM); err != nil {
					_ = os.Remove(pidFilePath)
					return fmt.Errorf("failed to stop service %s: %v", service, err)
				}
				_ = os.Remove(pidFilePath)
				fmt.Printf("Stopped service %s (pid %d) in directory %s\n", service, pid, dir)
				return nil
			},
		},
		{
			Name:  "logs",
			Usage: "Show logs for a service, or list available logs if no service is specified",
			Action: func(ctx *cli.Context) error {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home dir: %w", err)
				}
				logsDir := filepath.Join(homeDir, "micro", "logs")
				if ctx.Args().Len() == 0 {
					// List available logs
					dirEntries, err := os.ReadDir(logsDir)
					if err != nil {
						return fmt.Errorf("could not list logs directory: %v", err)
					}
					fmt.Println("Available logs:")
					found := false
					for _, entry := range dirEntries {
						if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
							fmt.Println("  ", strings.TrimSuffix(entry.Name(), ".log"))
							found = true
						}
					}
					if !found {
						fmt.Println("  (no logs found)")
					}
					return nil
				}
				service := ctx.Args().Get(0)
				logFilePath := filepath.Join(logsDir, service+".log")
				f, err := os.Open(logFilePath)
				if err != nil {
					return fmt.Errorf("could not open log file for service %s: %v", service, err)
				}
				defer f.Close()
				scan := bufio.NewScanner(f)
				for scan.Scan() {
					fmt.Println(scan.Text())
				}
				return scan.Err()
			},
		},
	}...)

	cmd.App().Action = func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			return nil
		}

		v, err := exec.LookPath("micro-" + c.Args().First())
		if err == nil {
			ce := exec.Command(v, c.Args().Slice()[1:]...)
			ce.Stdout = os.Stdout
			ce.Stderr = os.Stderr
			return ce.Run()
		}

		command := c.Args().Get(0)
		args := c.Args().Slice()

		if srv, err := util.LookupService(command); err != nil {
			return util.CliError(err)
		} else if srv != nil && util.ShouldRenderHelp(args) {
			return cli.Exit(util.FormatServiceUsage(srv, c), 0)
		} else if srv != nil {
			err := util.CallService(srv, args)
			return util.CliError(err)
		}

		return nil
	}
}
