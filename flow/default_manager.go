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

func (m *defaultManager) Register(*Flow, ...RegisterOption) error {
	return nil
}

func (m *defaultManager) Deregister(flow *Flow, opts ...DeregisterOption) error {
	return nil
}

func (m *defaultManager) Init(opts ...ManagerOption) error {
	return nil
}

func (m *defaultManager) Options() ManagerOptions {
	return m.options
}

func (m *defaultManager) Load(flow string) (*Flow, error) {
	return nil, nil
}

func (m *defaultManager) Save(flow *Flow) error {
	return nil
}
