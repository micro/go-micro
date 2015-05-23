package client

type RpcRequest struct {
	service     string
	method      string
	contentType string
	request     interface{}
}

func newRpcRequest(service, method string, request interface{}, contentType string) *RpcRequest {
	return &RpcRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: contentType,
	}
}

func (r *RpcRequest) ContentType() string {
	return r.contentType
}

func (r *RpcRequest) Service() string {
	return r.service
}

func (r *RpcRequest) Method() string {
	return r.method
}

func (r *RpcRequest) Request() interface{} {
	return r.request
}

func NewRpcRequest(service, method string, request interface{}, contentType string) *RpcRequest {
	return newRpcRequest(service, method, request, contentType)
}
