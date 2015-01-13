package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/asim/go-micro/errors"
	log "github.com/cihub/seelog"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"
)

type RpcServer struct {
	mtx     sync.RWMutex
	rpc     *rpc.Server
	address string
	exit    chan chan error
}

var (
	HealthPath = "/_status/health"
	RpcPath    = "/_rpc"
)

func executeRequestSafely(c *serverContext, r *http.Request) {
	defer func() {
		if x := recover(); x != nil {
			log.Criticalf("Panicked on request: %v", r)
			log.Criticalf("%v: %v", x, string(debug.Stack()))
			err := errors.InternalServerError("go.micro.server", "Unexpected error")
			c.WriteHeader(500)
			c.Write([]byte(err.Error()))
		}
	}()

	http.DefaultServeMux.ServeHTTP(c, r)
}

func (s *RpcServer) handler(w http.ResponseWriter, r *http.Request) {
	c := &serverContext{
		req:       &serverRequest{r},
		outHeader: w.Header(),
	}

	ctxs.Lock()
	ctxs.m[r] = c
	ctxs.Unlock()
	defer func() {
		ctxs.Lock()
		delete(ctxs.m, r)
		ctxs.Unlock()
	}()

	// Patch up RemoteAddr so it looks reasonable.
	if addr := r.Header.Get("X-Forwarded-For"); len(addr) > 0 {
		r.RemoteAddr = addr
	} else {
		// Should not normally reach here, but pick a sensible default anyway.
		r.RemoteAddr = "127.0.0.1"
	}
	// The address in the headers will most likely be of these forms:
	//	123.123.123.123
	//	2001:db8::1
	// net/http.Request.RemoteAddr is specified to be in "IP:port" form.
	if _, _, err := net.SplitHostPort(r.RemoteAddr); err != nil {
		// Assume the remote address is only a host; add a default port.
		r.RemoteAddr = net.JoinHostPort(r.RemoteAddr, "80")
	}

	executeRequestSafely(c, r)
	c.outHeader = nil // make sure header changes aren't respected any more

	// Avoid nil Write call if c.Write is never called.
	if c.outCode != 0 {
		w.WriteHeader(c.outCode)
	}
	if c.outBody != nil {
		w.Write(c.outBody)
	}
}

func (s *RpcServer) Address() string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.address
}

func (s *RpcServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	serveCtx := getServerContext(req)

	// TODO: get user scope from context
	// check access

	if req.Method != "POST" {
		err := errors.BadRequest("go.micro.server", "Method not allowed")
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		errr := errors.InternalServerError("go.micro.server", fmt.Sprintf("Error reading request body: %v", err))
		w.WriteHeader(500)
		w.Write([]byte(errr.Error()))
		log.Errorf("Erroring reading request body: %v", err)
		return
	}

	rbq := bytes.NewBuffer(b)
	rsp := bytes.NewBuffer(nil)
	defer rsp.Reset()
	defer rbq.Reset()

	buf := &buffer{
		rbq,
		rsp,
	}

	var cc rpc.ServerCodec
	switch req.Header.Get("Content-Type") {
	case "application/octet-stream":
		cc = pb.NewServerCodec(buf)
	case "application/json":
		cc = js.NewServerCodec(buf)
	default:
		err = errors.InternalServerError("go.micro.server", fmt.Sprintf("Unsupported content-type: %v", req.Header.Get("Content-Type")))
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	ctx := newContext(&ctx{}, serveCtx)
	err = s.rpc.ServeRequestWithContext(ctx, cc)
	if err != nil {
		// This should not be possible.
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		log.Errorf("Erroring serving request: %v", err)
		return
	}

	w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", strconv.Itoa(rsp.Len()))
	w.Write(rsp.Bytes())
}

func (s *RpcServer) Init() error {
	log.Debugf("Rpc handler %s", RpcPath)
	http.Handle(RpcPath, s)
	return nil
}

func (s *RpcServer) NewReceiver(handler interface{}) Receiver {
	return newRpcReceiver("", handler)
}

func (s *RpcServer) NewNamedReceiver(name string, handler interface{}) Receiver {
	return newRpcReceiver(name, handler)
}

func (s *RpcServer) Register(r Receiver) error {
	if len(r.Name()) > 0 {
		s.rpc.RegisterName(r.Name(), r.Handler())
		return nil
	}

	s.rpc.Register(r.Handler())
	return nil
}

func (s *RpcServer) Start() error {
	registerHealthChecker(http.DefaultServeMux)

	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	log.Debugf("Listening on %s", l.Addr().String())

	s.mtx.Lock()
	s.address = l.Addr().String()
	s.mtx.Unlock()

	go http.Serve(l, http.HandlerFunc(s.handler))

	go func() {
		ch := <-s.exit
		ch <- l.Close()
	}()

	return nil
}

func (s *RpcServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}

func NewRpcServer(address string) *RpcServer {
	return &RpcServer{
		rpc:     rpc.NewServer(),
		address: address,
		exit:    make(chan chan error),
	}
}
