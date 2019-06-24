// Package rpc is a go-micro rpc handler.
package rpc

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/joncalhoun/qson"
	"github.com/micro/go-micro/api"
	"github.com/micro/go-micro/api/handler"
	proto "github.com/micro/go-micro/api/internal/proto"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/jsonrpc"
	"github.com/micro/go-micro/codec/protorpc"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/util/ctx"
)

const (
	Handler = "rpc"
)

var (
	// supported json codecs
	jsonCodecs = []string{
		"application/grpc+json",
		"application/json",
		"application/json-rpc",
	}

	// support proto codecs
	protoCodecs = []string{
		"application/grpc",
		"application/grpc+proto",
		"application/proto",
		"application/protobuf",
		"application/proto-rpc",
		"application/octet-stream",
	}
)

type rpcHandler struct {
	opts handler.Options
	s    *api.Service
}

type buffer struct {
	io.ReadCloser
}

func (b *buffer) Write(_ []byte) (int, error) {
	return 0, nil
}

// strategy is a hack for selection
func strategy(services []*registry.Service) selector.Strategy {
	return func(_ []*registry.Service) selector.Next {
		// ignore input to this function, use services above
		return selector.Random(services)
	}
}

func (h *rpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var service *api.Service

	if h.s != nil {
		// we were given the service
		service = h.s
	} else if h.opts.Router != nil {
		// try get service from router
		s, err := h.opts.Router.Route(r)
		if err != nil {
			writeError(w, r, errors.InternalServerError("go.micro.api", err.Error()))
			return
		}
		service = s
	} else {
		// we have no way of routing the request
		writeError(w, r, errors.InternalServerError("go.micro.api", "no route found"))
		return
	}

	// only allow post when we have the router
	if r.Method != "GET" && (h.opts.Router != nil && r.Method != "POST") {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ct := r.Header.Get("Content-Type")

	// Strip charset from Content-Type (like `application/json; charset=UTF-8`)
	if idx := strings.IndexRune(ct, ';'); idx >= 0 {
		ct = ct[:idx]
	}

	// micro client
	c := h.opts.Service.Client()

	// create strategy
	so := selector.WithStrategy(strategy(service.Services))

	// get payload
	br, err := requestPayload(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// create context
	cx := ctx.FromRequest(r)

	var rsp []byte

	switch {
	// proto codecs
	case hasCodec(ct, protoCodecs):
		request := &proto.Message{}
		// if the extracted payload isn't empty lets use it
		if len(br) > 0 {
			request = proto.NewMessage(br)
		}

		// create request/response
		response := &proto.Message{}

		req := c.NewRequest(
			service.Name,
			service.Endpoint.Name,
			request,
			client.WithContentType(ct),
		)

		// make the call
		if err := c.Call(cx, req, response, client.WithSelectOption(so)); err != nil {
			writeError(w, r, err)
			return
		}

		// marshall response
		rsp, _ = response.Marshal()
	default:
		// if json codec is not present set to json
		if !hasCodec(ct, jsonCodecs) {
			ct = "application/json"
		}

		// default to trying json
		var request json.RawMessage
		// if the extracted payload isn't empty lets use it
		if len(br) > 0 {
			request = json.RawMessage(br)
		}

		// create request/response
		var response json.RawMessage

		req := c.NewRequest(
			service.Name,
			service.Endpoint.Name,
			&request,
			client.WithContentType(ct),
		)

		// make the call
		if err := c.Call(cx, req, &response, client.WithSelectOption(so)); err != nil {
			writeError(w, r, err)
			return
		}

		// marshall response
		rsp, _ = response.MarshalJSON()
	}

	// write the response
	writeResponse(w, r, rsp)
}

func (rh *rpcHandler) String() string {
	return "rpc"
}

func hasCodec(ct string, codecs []string) bool {
	for _, codec := range codecs {
		if ct == codec {
			return true
		}
	}
	return false
}

// requestPayload takes a *http.Request.
// If the request is a GET the query string parameters are extracted and marshaled to JSON and the raw bytes are returned.
// If the request method is a POST the request body is read and returned
func requestPayload(r *http.Request) ([]byte, error) {
	// we have to decode json-rpc and proto-rpc because we suck
	// well actually because there's no proxy codec right now
	switch r.Header.Get("Content-Type") {
	case "application/json-rpc":
		msg := codec.Message{
			Type:   codec.Request,
			Header: make(map[string]string),
		}
		c := jsonrpc.NewCodec(&buffer{r.Body})
		if err := c.ReadHeader(&msg, codec.Request); err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := c.ReadBody(&raw); err != nil {
			return nil, err
		}
		return ([]byte)(raw), nil
	case "application/proto-rpc", "application/octet-stream":
		msg := codec.Message{
			Type:   codec.Request,
			Header: make(map[string]string),
		}
		c := protorpc.NewCodec(&buffer{r.Body})
		if err := c.ReadHeader(&msg, codec.Request); err != nil {
			return nil, err
		}
		var raw proto.Message
		if err := c.ReadBody(&raw); err != nil {
			return nil, err
		}
		b, _ := raw.Marshal()
		return b, nil
	}

	// otherwise as per usual

	switch r.Method {
	case "GET":
		if len(r.URL.RawQuery) > 0 {
			return qson.ToJSON(r.URL.RawQuery)
		}
	case "PATCH", "POST":
		return ioutil.ReadAll(r.Body)
	}

	return []byte{}, nil
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	ce := errors.Parse(err.Error())

	switch ce.Code {
	case 0:
		// assuming it's totally screwed
		ce.Code = 500
		ce.Id = "go.micro.api"
		ce.Status = http.StatusText(500)
		ce.Detail = "error during request: " + ce.Detail
		w.WriteHeader(500)
	default:
		w.WriteHeader(int(ce.Code))
	}

	// response content type
	w.Header().Set("Content-Type", "application/json")

	// Set trailers
	if strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
		w.Header().Set("Trailer", "grpc-status")
		w.Header().Set("Trailer", "grpc-message")
		w.Header().Set("grpc-status", "13")
		w.Header().Set("grpc-message", ce.Detail)
	}

	w.Write([]byte(ce.Error()))
}

func writeResponse(w http.ResponseWriter, r *http.Request, rsp []byte) {
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", strconv.Itoa(len(rsp)))

	// Set trailers
	if strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
		w.Header().Set("Trailer", "grpc-status")
		w.Header().Set("Trailer", "grpc-message")
		w.Header().Set("grpc-status", "0")
		w.Header().Set("grpc-message", "")
	}

	// write response
	w.Write(rsp)
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.NewOptions(opts...)
	return &rpcHandler{
		opts: options,
	}
}

func WithService(s *api.Service, opts ...handler.Option) handler.Handler {
	options := handler.NewOptions(opts...)
	return &rpcHandler{
		opts: options,
		s:    s,
	}
}
