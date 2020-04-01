package rpc

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/micro/go-micro/v2/api"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	raw "github.com/micro/go-micro/v2/codec/bytes"
	"github.com/micro/go-micro/v2/logger"
)

// serveWebsocket will stream rpc back over websockets assuming json
func serveWebsocket(ctx context.Context, w http.ResponseWriter, r *http.Request, service *api.Service, c client.Client) {

	hdr := make(http.Header)
	//proto := r.Header.Get("Sec-WebSocket-Protocol")
	//fmt.Printf("%s\n", r.Header)
	//var wsproto string

	//	switch proto {
	//	case "binary":
	hdr["Sec-WebSocket-Protocol"] = []string{"binary"}
	//	default:
	// not allowed now
	//	}

	payload, err := requestPayload(r)
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(err)
		}
		return
	}

	upgrader := &ws.HTTPUpgrader{Timeout: 5 * time.Second,
		Protocol: func(proto string) bool {
			if strings.HasPrefix(proto, "Bearer") {
				return true
			}
			if strings.Contains(proto, "binary") {
				return true
			}
			return true
		},
		Extension: func(httphead.Option) bool {
			return false
		},
		Header: hdr,
	}

	conn, rw, _, err := upgrader.Upgrade(r, w)
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(err)
		}
		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
	}()

	// create stream before reading client data, because client
	// may want to wait server message first
	var req client.Request
	if len(payload) > 0 {
		//	request := json.RawMessage(payload)
		// create a request to the backend
		req = c.NewRequest(
			service.Name,
			service.Endpoint.Name,
			//			&request,
			&raw.Frame{},
			client.WithContentType("application/json"),
			client.StreamingRequest(),
		)
	} else {
		req = c.NewRequest(
			service.Name,
			service.Endpoint.Name,
			&raw.Frame{},
			client.StreamingRequest(),
		)
	}

	so := selector.WithStrategy(strategy(service.Services))
	// create a new stream
	stream, err := c.Stream(ctx, req, client.WithSelectOption(so))
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(err)
		}
		return
	}

	if len(payload) > 0 {
		// create a request to the backend
		request := &raw.Frame{Data: payload}
		if err = stream.Send(request); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
	}

	go writeLoop(rw, stream)

	rsp := stream.Response()

	// check proto from request
	op := ws.OpBinary

	// receive from stream and send to client
	for {
		// read backend response body
		buf, err := rsp.Read()
		if err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}

		// write the response
		if err := wsutil.WriteServerMessage(rw, op, buf); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
		if err = rw.Flush(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
	}
}

// writeLoop
func writeLoop(rw io.ReadWriter, stream client.Stream) {
	// close stream when done
	defer stream.Close()

	for {
		buf, op, err := wsutil.ReadClientData(rw)
		if err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
		switch op {
		default:
			// not relevant
			continue
		case ws.OpText, ws.OpBinary:
			break
		}

		// send to backend
		// default to trying json
		// if the extracted payload isn't empty lets use it
		request := &raw.Frame{Data: buf}

		if err := stream.Send(request); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
	}
}

func isStream(r *http.Request, srv *api.Service) bool {
	// check if it's a web socket
	if !isWebSocket(r) {
		return false
	}
	// check if the endpoint supports streaming
	for _, service := range srv.Services {
		for _, ep := range service.Endpoints {
			// skip if it doesn't match the name
			if ep.Name != srv.Endpoint.Name {
				continue
			}
			// matched if the name
			if v := ep.Metadata["stream"]; v == "true" {
				return true
			}
		}
	}
	return false
}

func isWebSocket(r *http.Request) bool {
	contains := func(key, val string) bool {
		vv := strings.Split(r.Header.Get(key), ",")
		for _, v := range vv {
			if val == strings.ToLower(strings.TrimSpace(v)) {
				return true
			}
		}
		return false
	}

	if contains("Connection", "upgrade") && contains("Upgrade", "websocket") {
		return true
	}

	return false
}
