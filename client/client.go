package client

type Client interface {
	NewRequest(string, string, interface{}) Request
	NewProtoRequest(string, string, interface{}) Request
	NewJsonRequest(string, string, interface{}) Request
	Call(interface{}, interface{}) error
	CallRemote(string, string, interface{}, interface{}) error
}

var (
	client = NewRpcClient()
)

func Call(request Request, response interface{}) error {
	return client.Call(request, response)
}

func CallRemote(address, path string, request Request, response interface{}) error {
	return client.CallRemote(address, path, request, response)
}

func NewRequest(service, method string, request interface{}) Request {
	return client.NewRequest(service, method, request)
}

func NewProtoRequest(service, method string, request interface{}) Request {
	return client.NewProtoRequest(service, method, request)
}

func NewJsonRequest(service, method string, request interface{}) Request {
	return client.NewJsonRequest(service, method, request)
}
