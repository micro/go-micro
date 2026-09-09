package server

import (
	"context"
	"io"

	"go-micro.dev/v6/internal/network"
	"go-micro.dev/v6/transport"
	"go-micro.dev/v6/transport/headers"
)

// local.go gives a same-process caller a way to reach this server's handlers
// without the network transport. A running server registers a dispatcher in
// internal/network keyed by its name; an opted-in client looks it up and
// calls localDispatch, which serves the request synchronously through the same
// router (so handler wrappers, codecs, and error mapping are identical) over an
// in-memory socket — skipping dial, the transport pump, and the codec-over-pipe
// double serialization. Unary only; streaming and pub/sub keep the normal path.

// localSocket is a transport.Socket that carries exactly one request in and
// captures exactly one reply — no network, no pipe, no gob. Recv delivers the
// request message once (the RPC codec reads it on the first ReadHeader), then
// reports EOF; Send captures the encoded reply.
type localSocket struct {
	req   *transport.Message
	recvd bool
	reply *transport.Message
}

func (s *localSocket) Recv(m *transport.Message) error {
	if s.recvd || s.req == nil {
		return io.EOF
	}
	s.recvd = true
	m.Header = s.req.Header
	m.Body = s.req.Body
	return nil
}

func (s *localSocket) Send(m *transport.Message) error {
	cp := &transport.Message{Header: make(map[string]string, len(m.Header))}
	for k, v := range m.Header {
		cp.Header[k] = v
	}
	if len(m.Body) > 0 {
		cp.Body = append([]byte(nil), m.Body...)
	}
	s.reply = cp
	return nil
}

func (s *localSocket) Close() error   { return nil }
func (s *localSocket) Local() string  { return "local" }
func (s *localSocket) Remote() string { return "local" }

// localDispatch serves req against this server's router in-process and returns
// the reply. It mirrors the request/response construction ServeConn does for a
// networked request, so the served path is identical apart from the transport.
func (s *rpcServer) localDispatch(ctx context.Context, req *transport.Message) (*transport.Message, error) {
	contentType := req.Header["Content-Type"]
	if contentType == "" {
		contentType = DefaultContentType
		req.Header["Content-Type"] = contentType
	}

	cf := setupProtocol(req)
	if cf == nil {
		var err error
		if cf, err = s.newCodec(contentType); err != nil {
			return nil, err
		}
	}

	sock := &localSocket{req: req}
	rcodec := newRPCCodec(req, sock, cf)

	request := rpcRequest{
		service:     getHeader(headers.Request, req.Header),
		method:      getHeader(headers.Method, req.Header),
		endpoint:    getHeader(headers.Endpoint, req.Header),
		contentType: contentType,
		codec:       rcodec,
		header:      req.Header,
		body:        req.Body,
		socket:      sock,
	}
	response := rpcResponse{
		header: make(map[string]string),
		socket: sock,
		codec:  rcodec,
	}

	if err := s.getRouter().ServeRequest(ctx, &request, &response); err != nil {
		return nil, err
	}
	if sock.reply == nil {
		// A handler that wrote no body still completed successfully.
		return &transport.Message{Header: map[string]string{}}, nil
	}
	return sock.reply, nil
}

// registerLocal makes this server reachable in-process under its name; called
// on Start. deregisterLocal removes it on Stop.
func (s *rpcServer) registerLocal() {
	name := s.Options().Name
	if name == "" {
		return
	}
	network.Register(name, s.localDispatch)
}

func (s *rpcServer) deregisterLocal() {
	name := s.Options().Name
	if name == "" {
		return
	}
	network.Deregister(name)
}
