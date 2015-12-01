package client

type rpcStream struct {
	request Request
	call    *call
	client  *client
}

func (r *rpcStream) Request() Request {
	return r.request
}

func (r *rpcStream) Error() error {
	return r.call.Error
}

func (r *rpcStream) Close() error {
	return r.client.Close()
}
