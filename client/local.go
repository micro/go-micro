package client

import (
	"context"

	raw "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/internal/network"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/transport"
	"go-micro.dev/v6/transport/headers"
)

// localCall is the in-process fast-path for Call. When LocalDispatch is enabled
// and the callee runs in this same process, a unary request whose body and
// response are raw frames (codec/bytes.Frame) is dispatched straight to the
// server's handlers via internal/network — no dial, no codec-over-socket,
// no transport pump. It returns handled=false to fall back to the network path
// for anything it does not cover (disabled, streaming, non-frame bodies, or a
// service not registered in-process), so behavior is unchanged unless the
// fast-path fully applies.
func (r *rpcClient) localCall(ctx context.Context, req Request, resp interface{}) (handled bool, err error) {
	if !r.opts.LocalDispatch || req.Stream() {
		return false, nil
	}
	reqFrame, ok := req.Body().(*raw.Frame)
	if !ok {
		return false, nil
	}
	respFrame, ok := resp.(*raw.Frame)
	if !ok {
		return false, nil
	}
	dispatch, ok := network.Lookup(req.Service())
	if !ok {
		return false, nil
	}

	header := make(map[string]string)
	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			if k == headers.Message { // pub/sub topic header, never forwarded
				continue
			}
			header[k] = v
		}
	}
	header[headers.Request] = req.Service()
	header[headers.Endpoint] = req.Endpoint()
	header["Content-Type"] = req.ContentType()
	header["Accept"] = req.ContentType()

	reply, err := dispatch(ctx, &transport.Message{Header: header, Body: reqFrame.Data})
	if err != nil {
		return true, err
	}
	if reply != nil {
		respFrame.Data = reply.Body
	}
	return true, nil
}
