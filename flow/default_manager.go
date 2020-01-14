package flow

import "sync"

type defaultManager struct {
	sync.RWMutex
	options ManagerOptions
}

// Create default manager
func NewManager(opts ...ManagerOption) Manager {
	options := ManagerOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	mgr := &defaultManager{
		options: options,
	}

	return mgr
}

func (mgr *defaultManager) Register(flow *Flow, opts ...RegisterOption) error {
	return nil
}

func (mgr *defaultManager) Deregister(flow *Flow, opts ...DeregisterOption) error {
	return nil
}

func (mgr *defaultManager) Init(opts ...ManagerOption) error {
	for _, opt := range opts {
		opt(&mgr.options)
	}

	return nil
}

func (mgr *defaultManager) Options() ManagerOptions {
	return mgr.options
}
