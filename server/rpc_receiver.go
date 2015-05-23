package server

type rpcReceiver struct {
	name    string
	handler interface{}
}

func newRpcReceiver(name string, handler interface{}) Receiver {
	return &rpcReceiver{
		name:    name,
		handler: handler,
	}
}

func (r *rpcReceiver) Name() string {
	return r.name
}

func (r *rpcReceiver) Handler() interface{} {
	return r.handler
}
