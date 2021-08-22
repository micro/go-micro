// Package pprof provides a pprof profiler
package pprof

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/asim/go-micro/v3/debug/profile"
)

type profiler struct {
	opts profile.Options

	sync.Mutex
	running bool

	// where the cpu profile is written
	cpuFile *os.File
	// where the mem profile is written
	memFile *os.File
}

func (p *profiler) Start() error {
	p.Lock()
	defer p.Unlock()

	if p.running {
		return nil
	}

	cpuFile := filepath.Join(os.TempDir(), "cpu.pprof")
	memFile := filepath.Join(os.TempDir(), "mem.pprof")

	if len(p.opts.Name) > 0 {
		cpuFile = filepath.Join(os.TempDir(), p.opts.Name+".cpu.pprof")
		memFile = filepath.Join(os.TempDir(), p.opts.Name+".mem.pprof")
	}

	f1, err := os.Create(cpuFile)
	if err != nil {
		return err
	}

	f2, err := os.Create(memFile)
	if err != nil {
		return err
	}

	// start cpu profiling
	if err := pprof.StartCPUProfile(f1); err != nil {
		return err
	}

	// set cpu file
	p.cpuFile = f1
	// set mem file
	p.memFile = f2

	p.running = true

	return nil
}

func (p *profiler) Stop() error {
	p.Lock()
	defer p.Unlock()

	if !p.running {
		return nil
	}

	pprof.StopCPUProfile()
	p.cpuFile.Close()
	runtime.GC()
	pprof.WriteHeapProfile(p.memFile)
	p.memFile.Close()
	p.running = false
	p.cpuFile = nil
	p.memFile = nil
	return nil
}

func (p *profiler) String() string {
	return "pprof"
}

func NewProfile(opts ...profile.Option) profile.Profile {
	var options profile.Options
	for _, o := range opts {
		o(&options)
	}
	p := new(profiler)
	p.opts = options
	return p
}
