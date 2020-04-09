// +build !windows

// Package os runs processes locally
package os

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/micro/go-micro/v2/runtime/local/process"
)

func (p *Process) Exec(exe *process.Executable) error {
	cmd := exec.Command(exe.Package.Path)
	cmd.Dir = exe.Dir
	return cmd.Run()
}

func (p *Process) Fork(exe *process.Executable) (*process.PID, error) {
	// create command
	cmd := exec.Command(exe.Package.Path, exe.Args...)

	cmd.Dir = exe.Dir
	// set env vars
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, exe.Env...)

	// create process group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	er, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// start the process
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &process.PID{
		ID:     fmt.Sprintf("%d", cmd.Process.Pid),
		Input:  in,
		Output: out,
		Error:  er,
	}, nil
}

func (p *Process) Kill(pid *process.PID) error {
	id, err := strconv.Atoi(pid.ID)
	if err != nil {
		return err
	}

	pr, err := os.FindProcess(id)
	if err != nil {
		return err
	}

	// now kill it
	err = pr.Kill()

	// kill the group
	if pgid, err := syscall.Getpgid(id); err == nil {
		syscall.Kill(-pgid, syscall.SIGKILL)
	}

	// return the kill error
	return err
}

func (p *Process) Wait(pid *process.PID) error {
	id, err := strconv.Atoi(pid.ID)
	if err != nil {
		return err
	}

	pr, err := os.FindProcess(id)
	if err != nil {
		return err
	}

	ps, err := pr.Wait()
	if err != nil {
		return err
	}

	if ps.Success() {
		return nil
	}

	return fmt.Errorf(ps.String())
}
