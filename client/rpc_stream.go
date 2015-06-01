package client

import (
	rpc "github.com/youtube/vitess/go/rpcplus"
)

type rpcStream struct {
	request Request
	call    *rpc.Call
	client  *rpc.Client
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
