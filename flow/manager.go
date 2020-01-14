package flow

type Manager interface {
	Init(...ManagerOption) error
	Options() ManagerOptions
	Register(*Flow, ...RegisterOption) error
	Deregister(*Flow, ...DeregisterOption) error
}

type Flow struct {
	Name       string
	Operations []Operation
}

type ManagerOption func(*ManagerOptions)

func ManagerFlowStore(s FlowStore) ManagerOption {
	return func(o *ManagerOptions) {
		o.FlowStore = s
	}
}

type ManagerOptions struct {
	FlowStore FlowStore
}

type RegisterOptions struct {
	Append   bool
	Requires []string
	Required []string
}

type DeregisterOptions struct {
	Partial bool
}

type RegisterOption func(*RegisterOptions)

type DeregisterOption func(*DeregisterOptions)

func RegisterRequires(req []string) RegisterOption {
	return func(o *RegisterOptions) {
		o.Requires = req
	}
}

func RegisterRequired(req []string) RegisterOption {
	return func(o *RegisterOptions) {
		o.Required = req
	}
}

func RegisterAppend(b bool) RegisterOption {
	return func(o *RegisterOptions) {
		o.Append = b
	}
}
