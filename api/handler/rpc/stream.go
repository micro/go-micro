package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go-micro.dev/v4/api/router"
	"go-micro.dev/v4/client"
	raw "go-micro.dev/v4/codec/bytes"
	"go-micro.dev/v4/selector"
)

// serveWebsocket will stream rpc back over websockets assuming json.
func serveWebsocket(ctx context.Context, w http.ResponseWriter, r *http.Request, service *router.Route, c client.Client) (err error) {
	var opCode ws.OpCode

	myCt := r.Header.Get("Content-Type")
	// Strip charset from Content-Type (like `application/json; charset=UTF-8`)
	if idx := strings.IndexRune(myCt, ';'); idx >= 0 {
		myCt = myCt[:idx]
	}

	// check proto from request
	switch myCt {
	case "application/json":
		opCode = ws.OpText
	default:
		opCode = ws.OpBinary
	}

	hdr := make(http.Header)

	if proto, ok := r.Header["Sec-Websocket-Protocol"]; ok {
		for _, p := range proto {
			if p == "binary" {
				hdr["Sec-WebSocket-Protocol"] = []string{"binary"}
				opCode = ws.OpBinary
			}
		}
	}

	payload, err := requestPayload(r)
	if err != nil {
		return
	}

	upgrader := ws.HTTPUpgrader{Timeout: 5 * time.Second,
		Protocol: func(proto string) bool {
			if strings.Contains(proto, "binary") {
				return true
			}
			// fallback to support all protocols now
			return true
		},
		Extension: func(httphead.Option) bool {
			// disable extensions for compatibility
			return false
		},
		Header: hdr,
	}

	conn, uRw, _, err := upgrader.Upgrade(r, w)
	if err != nil {
		return
	}

	defer func() {
		if cErr := conn.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	var request interface{}

	if !bytes.Equal(payload, []byte(`{}`)) {
		switch myCt {
		case "application/json", "":
			m := json.RawMessage(payload)
			request = &m
		default:
			request = &raw.Frame{Data: payload}
		}
	}

	// we always need to set content type for message
	if myCt == "" {
		myCt = "application/json"
	}

	req := c.NewRequest(
		service.Service,
		service.Endpoint.Name,
		request,
		client.WithContentType(myCt),
		client.StreamingRequest(),
	)

	so := selector.WithStrategy(strategy(service.Versions))
	// create a new stream
	stream, err := c.Stream(ctx, req, client.WithSelectOption(so))
	if err != nil {
		return
	}

	if request != nil {
		if err = stream.Send(request); err != nil {
			return
		}
	}

	go func() {
		if wErr := writeLoop(uRw, stream); wErr != nil && err == nil {
			err = wErr
		}
	}()

	rsp := stream.Response()

	// receive from stream and send to client
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-stream.Context().Done():
			return nil
		default:
			// read backend response body
			buf, err := rsp.Read()
			if err != nil {
				// wants to avoid import  grpc/status.Status
				if strings.Contains(err.Error(), "context canceled") {
					return nil
				}

				return err
			}

			// write the response
			if err = wsutil.WriteServerMessage(uRw, opCode, buf); err != nil {
				return err
			}

			if err = uRw.Flush(); err != nil {
				return err
			}
		}
	}
}

// writeLoop.
func writeLoop(rw io.ReadWriter, stream client.Stream) error {
	// close stream when done
	defer stream.Close()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			buf, op, err := wsutil.ReadClientData(rw)
			if err != nil {
				if wserr, ok := err.(wsutil.ClosedError); ok {
					switch wserr.Code {
					case ws.StatusGoingAway:
						// this happens when user leave the page
						return nil
					case ws.StatusNormalClosure, ws.StatusNoStatusRcvd:
						// this happens when user close ws connection, or we don't get any status
						return nil
					default:
						return err
					}
				}
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
				return err
			}
		}
	}
}

func isStream(r *http.Request, srv *router.Route) bool {
	// check if it's a web socket
	if !isWebSocket(r) {
		return false
	}
	// check if the endpoint supports streaming
	for _, service := range srv.Versions {
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
