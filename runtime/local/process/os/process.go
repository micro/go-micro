// Package os runs processes locally
package os

import (
	"go-micro.dev/v4/runtime/local/process"
)

type Process struct{}

func NewProcess(opts ...process.Option) process.Process {
	return &Process{}
}
