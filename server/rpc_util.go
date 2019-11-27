package server

import (
	"sync"
)

// waitgroup for global management of connections
type waitGroup struct {
	// local waitgroup
	lg sync.WaitGroup
	// global waitgroup
	gg *sync.WaitGroup
}

func (w *waitGroup) Add(i int) {
	w.lg.Add(i)
	if w.gg != nil {
		w.gg.Add(i)
	}
}

func (w *waitGroup) Done() {
	w.lg.Done()
	if w.gg != nil {
		w.gg.Done()
	}
}

func (w *waitGroup) Wait() {
	// only wait on local group
	w.lg.Wait()
}
