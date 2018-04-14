package client

type rpcRequest struct {
	service     string
	method      string
	contentType string
	request     interface{}
	opts        RequestOptions
}

func newRequest(service, method string, request interface{}, contentType string, reqOpts ...RequestOption) Request {
	var opts RequestOptions

	for _, o := range reqOpts {
		o(&opts)
	}

	// set the content-type specified
	if len(opts.ContentType) > 0 {
		contentType = opts.ContentType
	}

	return &rpcRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: contentType,
		opts:        opts,
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

func (r *rpcRequest) Stream() bool {
	return r.opts.Stream
}
