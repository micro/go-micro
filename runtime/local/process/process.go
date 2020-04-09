// Package process executes a binary
package process

import (
	"io"

	"github.com/micro/go-micro/v2/runtime/local/build"
)

// Process manages a running process
type Process interface {
	// Executes a process to completion
	Exec(*Executable) error
	// Creates a new process
	Fork(*Executable) (*PID, error)
	// Kills the process
	Kill(*PID) error
	// Waits for a process to exit
	Wait(*PID) error
}

type Executable struct {
	// Package containing executable
	Package *build.Package
	// The env variables
	Env []string
	// Args to pass
	Args []string
	// Initial working directory
	Dir string
}

// PID is the running process
type PID struct {
	// ID of the process
	ID string
	// Stdin
	Input io.Writer
	// Stdout
	Output io.Reader
	// Stderr
	Error io.Reader
}
