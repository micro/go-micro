package server

type RpcReceiver struct {
	name    string
	handler interface{}
}

func newRpcReceiver(name string, handler interface{}) *RpcReceiver {
	return &RpcReceiver{
		name:    name,
		handler: handler,
	}
}

func (r *RpcReceiver) Name() string {
	return r.name
}

func (r *RpcReceiver) Handler() interface{} {
	return r.handler
}

func NewRpcReceiver(handler interface{}) *RpcReceiver {
	return newRpcReceiver("", handler)
}

func NewNamedRpcReceiver(name string, handler interface{}) *RpcReceiver {
	return newRpcReceiver(name, handler)
}
