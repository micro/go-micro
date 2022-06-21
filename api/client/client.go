// Package client provides an api client
package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// local address for api
	localAddress = "http://localhost:8080"
	// public address for api
	liveAddress = "https://api.m3o.com"
)

// Options of the Client
type Options struct {
	// Token for authentication
	Token string
	// Address of the micro platform.
	// By default it connects to live. Change it or use the local flag
	// to connect to your local installation.
	Address string
	// Helper flag to help users connect to the default local address
	Local bool
	// set a timeout
	Timeout time.Duration
}

// Request is the request of the generic `api-client` call
type Request struct {
	// eg. "go.micro.srv.greeter"
	Service string `json:"service"`
	// eg. "Say.Hello"
	Endpoint string `json:"endpoint"`
	// json and then base64 encoded body
	Body string `json:"body"`
}

// Response is the response of the generic `api-client` call.
type Response struct {
	// json and base64 encoded response body
	Body string `json:"body"`
	// error fields. Error json example
	// {"id":"go.micro.client","code":500,"detail":"malformed method name: \"\"","status":"Internal Server Error"}
	Code   int    `json:"code"`
	ID     string `json:"id"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}

// Client enables generic calls to micro
type Client struct {
	options Options
}

type Stream struct {
	conn              *websocket.Conn
	service, endpoint string
}

// NewClient returns a generic micro client that connects to live by default
func NewClient(options *Options) *Client {
	ret := new(Client)
	ret.options = Options{
		Address: liveAddress,
	}

	// no options provided
	if options == nil {
		return ret
	}

	if options.Token != "" {
		ret.options.Token = options.Token
	}

	if options.Local {
		ret.options.Address = localAddress
		ret.options.Local = true
	}

	if options.Timeout > 0 {
		ret.options.Timeout = options.Timeout
	}

	return ret
}

// SetToken sets the api auth token
func (client *Client) SetToken(t string) {
	client.options.Token = t
}

// SetTimeout sets the http client's timeout
func (client *Client) SetTimeout(d time.Duration) {
	client.options.Timeout = d
}

// Call enables you to access any endpoint of any service on Micro
func (client *Client) Call(service, endpoint string, request, response interface{}) error {
	// example curl: curl -XPOST -d '{"service": "go.micro.srv.greeter", "endpoint": "Say.Hello"}'
	//  -H 'Content-Type: application/json' http://localhost:8080/client {"body":"eyJtc2ciOiJIZWxsbyAifQ=="}
	uri, err := url.Parse(client.options.Address)
	if err != nil {
		return err
	}

	// set the url to go through the v1 api
	uri.Path = "/v1/" + service + "/" + endpoint

	b, err := marshalRequest(service, endpoint, request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uri.String(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	// set the token if it exists
	if len(client.options.Token) > 0 {
		req.Header.Set("Authorization", "Bearer "+client.options.Token)
	}

	req.Header.Set("Content-Type", "application/json")

	// if user didn't specify Timeout the default is 0 i.e no timeout
	httpClient := &http.Client{
		Timeout: client.options.Timeout,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return errors.New(string(body))
	}
	return unmarshalResponse(body, response)
}

// Stream enables the ability to stream via websockets
func (client *Client) Stream(service, endpoint string, request interface{}) (*Stream, error) {
	b, err := marshalRequest(service, endpoint, request)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(client.options.Address)
	if err != nil {
		return nil, err
	}

	// set the url to go through the v1 api
	uri.Path = "/v1/" + service + "/" + endpoint

	// replace http with websocket
	uri.Scheme = strings.Replace(uri.Scheme, "http", "ws", 1)

	// create the headers
	header := make(http.Header)
	// set the token if it exists
	if len(client.options.Token) > 0 {
		header.Set("Authorization", "Bearer "+client.options.Token)
	}
	header.Set("Content-Type", "application/json")

	// dial the connection
	conn, _, err := websocket.DefaultDialer.Dial(uri.String(), header)
	if err != nil {
		return nil, err
	}

	// send the first request
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return nil, err
	}

	return &Stream{conn, service, endpoint}, nil
}

func (s *Stream) Recv(v interface{}) error {
	// read response
	_, message, err := s.conn.ReadMessage()
	if err != nil {
		return err
	}
	return unmarshalResponse(message, v)
}

func (s *Stream) Send(v interface{}) error {
	b, err := marshalRequest(s.service, s.endpoint, v)
	if err != nil {
		return err
	}
	return s.conn.WriteMessage(websocket.TextMessage, b)
}

func marshalRequest(service, endpoint string, v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshalResponse(body []byte, v interface{}) error {
	return json.Unmarshal(body, &v)
}
