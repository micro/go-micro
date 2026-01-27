package run

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/cmd/micro/run/config"
	"go-micro.dev/v5/cmd/micro/run/watcher"
)

// Color codes for log output
var colors = []string{
	"\033[31m", // red
	"\033[32m", // green
	"\033[33m", // yellow
	"\033[34m", // blue
	"\033[35m", // magenta
	"\033[36m", // cyan
}

const colorReset = "\033[0m"

func colorFor(idx int) string {
	return colors[idx%len(colors)]
}

// serviceProcess tracks a running service
type serviceProcess struct {
	name       string
	dir        string
	binPath    string
	pidFile    string
	logFile    string
	cmd        *exec.Cmd
	pipeWriter *io.PipeWriter
	color      string
	port       int
	env        []string

	mu      sync.Mutex
	running bool
}

func (s *serviceProcess) start(logDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Build
	buildCmd := exec.Command("go", "build", "-o", s.binPath, ".")
	buildCmd.Dir = s.dir
	buildOut, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		return fmt.Errorf("build failed: %s\n%s", buildErr, string(buildOut))
	}

	// Open log file
	logFile, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Start process
	s.cmd = exec.Command(s.binPath)
	s.cmd.Dir = s.dir
	s.cmd.Env = append(os.Environ(), s.env...)

	pr, pw := io.Pipe()
	s.pipeWriter = pw
	s.cmd.Stdout = pw
	s.cmd.Stderr = pw

	// Stream output
	go func(name string, color string, pr *io.PipeReader, logFile *os.File) {
		defer logFile.Close()
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("%s[%s]%s %s\n", color, name, colorReset, line)
			logFile.WriteString("[" + name + "] " + line + "\n")
		}
	}(s.name, s.color, pr, logFile)

	if err := s.cmd.Start(); err != nil {
		pw.Close()
		return fmt.Errorf("failed to start: %w", err)
	}

	// Write PID file
	os.WriteFile(s.pidFile, []byte(fmt.Sprintf("%d\n%s\n%s\n%s\n",
		s.cmd.Process.Pid, s.dir, s.name, time.Now().Format(time.RFC3339))), 0644)

	s.running = true
	fmt.Printf("%s[%s]%s started (pid %d)\n", s.color, s.name, colorReset, s.cmd.Process.Pid)

	return nil
}

func (s *serviceProcess) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.cmd == nil || s.cmd.Process == nil {
		return
	}

	fmt.Printf("%s[%s]%s stopping...\n", s.color, s.name, colorReset)

	// Graceful shutdown
	s.cmd.Process.Signal(syscall.SIGTERM)

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.cmd.Process.Kill()
		<-done
	}

	if s.pipeWriter != nil {
		s.pipeWriter.Close()
	}

	os.Remove(s.pidFile)
	s.running = false
}

func (s *serviceProcess) restart(logDir string) error {
	s.stop()
	return s.start(logDir)
}

// waitForHealth waits for a service's health endpoint to respond
func waitForHealth(port int, timeout time.Duration) bool {
	if port == 0 {
		return true // No port configured, assume ready
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func Run(c *cli.Context) error {
	dir := c.Args().Get(0)
	if dir == "" {
		dir = "."
	}

	// Handle git URLs
	if strings.HasPrefix(dir, "github.com/") || strings.HasPrefix(dir, "https://github.com/") {
		repo := strings.TrimPrefix(dir, "https://")
		tmp, err := os.MkdirTemp("", "micro-run-")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmp)

		cloneURL := "https://" + repo
		cloneCmd := exec.Command("git", "clone", "--depth", "1", cloneURL, tmp)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("failed to clone %s: %w", cloneURL, err)
		}
		dir = tmp
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Setup directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	logsDir := filepath.Join(homeDir, "micro", "logs")
	runDir := filepath.Join(homeDir, "micro", "run")
	binDir := filepath.Join(homeDir, "micro", "bin")

	for _, d := range []string{logsDir, runDir, binDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", d, err)
		}
	}

	// Load configuration
	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get environment
	envName := c.String("env")
	if envName == "" {
		envName = os.Getenv("MICRO_ENV")
	}
	if envName == "" {
		envName = "development"
	}

	var envVars []string
	if cfg != nil {
		if envMap := cfg.GetEnv(envName); envMap != nil {
			for k, v := range envMap {
				envVars = append(envVars, k+"="+v)
			}
		}
	}

	// Discover services
	var services []*serviceProcess
	servicesByDir := make(map[string]*serviceProcess)

	if cfg != nil && len(cfg.Services) > 0 {
		// Use configured services in dependency order
		sorted, err := cfg.TopologicalSort()
		if err != nil {
			return fmt.Errorf("dependency error: %w", err)
		}

		for i, svc := range sorted {
			svcDir := filepath.Join(absDir, svc.Path)
			absSvcDir, _ := filepath.Abs(svcDir)
			hash := fmt.Sprintf("%x", md5.Sum([]byte(absSvcDir)))[:8]

			sp := &serviceProcess{
				name:    svc.Name,
				dir:     absSvcDir,
				binPath: filepath.Join(binDir, svc.Name+"-"+hash),
				pidFile: filepath.Join(runDir, svc.Name+"-"+hash+".pid"),
				logFile: filepath.Join(logsDir, svc.Name+"-"+hash+".log"),
				color:   colorFor(i),
				port:    svc.Port,
				env:     envVars,
			}
			services = append(services, sp)
			servicesByDir[absSvcDir] = sp
		}
	} else {
		// Auto-discover from main.go files
		var mainFiles []string
		filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if info.Name() == "main.go" {
				mainFiles = append(mainFiles, path)
			}
			return nil
		})

		if len(mainFiles) == 0 {
			return fmt.Errorf("no main.go files found in %s", absDir)
		}

		for i, mainFile := range mainFiles {
			svcDir := filepath.Dir(mainFile)
			absSvcDir, _ := filepath.Abs(svcDir)

			var name string
			if absSvcDir == absDir {
				name = filepath.Base(absDir)
			} else {
				name = filepath.Base(svcDir)
			}

			hash := fmt.Sprintf("%x", md5.Sum([]byte(absSvcDir)))[:8]

			sp := &serviceProcess{
				name:    name,
				dir:     absSvcDir,
				binPath: filepath.Join(binDir, name+"-"+hash),
				pidFile: filepath.Join(runDir, name+"-"+hash+".pid"),
				logFile: filepath.Join(logsDir, name+"-"+hash+".log"),
				color:   colorFor(i),
				env:     envVars,
			}
			services = append(services, sp)
			servicesByDir[absSvcDir] = sp
		}
	}

	if len(services) == 0 {
		return fmt.Errorf("no services found")
	}

	// Start services
	fmt.Printf("Starting %d service(s)...\n", len(services))
	for _, svc := range services {
		if err := svc.start(logsDir); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] %v\n", svc.name, err)
			continue
		}

		// Wait for health if port configured
		if svc.port > 0 {
			if !waitForHealth(svc.port, 10*time.Second) {
				fmt.Fprintf(os.Stderr, "[%s] health check timeout\n", svc.name)
			}
		}
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Watch mode
	watchEnabled := !c.Bool("no-watch")
	var watch *watcher.Watcher

	if watchEnabled {
		var dirs []string
		for _, svc := range services {
			dirs = append(dirs, svc.dir)
		}

		watch = watcher.New(dirs)
		watch.Start()
		fmt.Println("Watching for changes... (use --no-watch to disable)")

		go func() {
			for event := range watch.Events() {
				if svc, ok := servicesByDir[event.Dir]; ok {
					fmt.Printf("%s[%s]%s rebuilding...\n", svc.color, svc.name, colorReset)
					if err := svc.restart(logsDir); err != nil {
						fmt.Fprintf(os.Stderr, "%s[%s]%s restart failed: %v\n", svc.color, svc.name, colorReset, err)
					}
				}
			}
		}()
	}

	// Wait for signal
	<-sigCh
	fmt.Println("\nShutting down...")

	if watch != nil {
		watch.Stop()
	}

	// Stop services in reverse order
	for i := len(services) - 1; i >= 0; i-- {
		services[i].stop()
	}

	return nil
}

// Helper functions
func parsePid(pidStr string) int {
	pid, _ := strconv.Atoi(pidStr)
	return pid
}

func processRunning(pidStr string) bool {
	pid := parsePid(pidStr)
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "run",
		Usage: "Run services with hot reload",
		Description: `Run discovers and runs services in a directory.

With a micro.mu or micro.json config file, services start in dependency order.
Without config, all main.go files are discovered and run.

Examples:
  micro run                    # Run services in current directory with hot reload
  micro run ./myapp            # Run services in ./myapp
  micro run --no-watch         # Run without hot reload
  micro run --env production   # Use production environment
  micro run github.com/micro/blog  # Clone and run`,
		Action: Run,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-watch",
				Usage: "Disable hot reload (file watching)",
			},
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment to use (default: development)",
				EnvVars: []string{"MICRO_ENV"},
			},
		},
	})
}
