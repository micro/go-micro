// +build windows

package run

import (
	"os"
	"os/exec"
	"syscall"
)

// Service is the interface that wraps the service that should run.
//
// Start starts the service and exits on error.
//
// Stop stops the service and exits on error.
//
// Wait waits for the service to exit and exits on error.
type Service interface {
	Start() error
	Stop() error
	Wait() error
}

type service struct {
	cmd     *exec.Cmd
	running bool
}

// Start starts the service and exits on error.
func (s *service) Start() error {
	if s.running {
		return nil
	}

	s.cmd = exec.Command("go", "run", ".")
	s.cmd.SysProcAttr = &syscall.SysProcAttr{}
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	s.running = true

	return s.cmd.Start()
}

// Stop stops the service and exits on error.
func (s *service) Stop() error {
	if !s.running {
		return nil
	}

	s.running = false
	pro, err := os.FindProcess(s.cmd.Process.Pid)
	if err != nil {
		return err
	}
	return pro.Signal(syscall.SIGTERM)
}

// Wait waits for the service to exit and exits on error.
func (s *service) Wait() error {
	return s.cmd.Wait()
}

func newService() *service {
	service := new(service)
	return service
}
