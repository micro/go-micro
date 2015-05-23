package client

type rpcRequest struct {
	service     string
	method      string
	contentType string
	request     interface{}
}

func newRpcRequest(service, method string, request interface{}, contentType string) Request {
	return &rpcRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: contentType,
	}
}

func (r *rpcRequest) ContentType() string {
	return r.contentType
}

func (r *rpcRequest) Service() string {
	return r.service
}

func (r *rpcRequest) Method() string {
	return r.method
}

func (r *rpcRequest) Request() interface{} {
	return r.request
}
