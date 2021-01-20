// Package os runs processes locally
package os

import (
	"github.com/asim/go-micro/v3/runtime/local/process"
)

type Process struct{}

func NewProcess(opts ...process.Option) process.Process {
	return &Process{}
}
