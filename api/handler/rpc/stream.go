package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/micro/go-micro/v2/api"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// serveWebsocket will stream rpc back over websockets assuming json
func serveWebsocket(ctx context.Context, w http.ResponseWriter, r *http.Request, service *api.Service, c client.Client) {
	// upgrade the connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	// close on exit
	defer conn.Close()

	// wait for the first request so we know
	_, p, err := conn.ReadMessage()
	if err != nil {
		return
	}

	// send to backend
	// default to trying json
	var request json.RawMessage
	// if the extracted payload isn't empty lets use it
	if len(p) > 0 {
		request = json.RawMessage(p)
	}

	// create a request to the backend
	req := c.NewRequest(
		service.Name,
		service.Endpoint.Name,
		&request,
		client.WithContentType("application/json"),
	)

	so := selector.WithStrategy(strategy(service.Services))

	// create a new stream
	stream, err := c.Stream(ctx, req, client.WithSelectOption(so))
	if err != nil {
		return
	}

	// send the first request for the client
	// since
	if err := stream.Send(request); err != nil {
		return
	}

	go writeLoop(conn, stream)

	resp := stream.Response()

	// receive from stream and send to client
	for {
		// read backend response body
		body, err := resp.Read()
		if err != nil {
			return
		}

		// write the response
		if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
			return
		}
	}
}

// writeLoop
func writeLoop(conn *websocket.Conn, stream client.Stream) {
	// close stream when done
	defer stream.Close()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// send to backend
		// default to trying json
		var request json.RawMessage
		// if the extracted payload isn't empty lets use it
		if len(p) > 0 {
			request = json.RawMessage(p)
		}

		if err := stream.Send(request); err != nil {
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
