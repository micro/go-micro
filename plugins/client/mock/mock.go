// Package mock provides a mock client for testing
package mock

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/errors"
)

var (
	_ client.Client = NewClient()
)

type MockResponse struct {
	Endpoint string
	Response interface{}
	Error    error
}

type MockClient struct {
	Client client.Client
	Opts   client.Options

	sync.Mutex
	Response map[string][]MockResponse
}

func (m *MockClient) Init(opts ...client.Option) error {
	m.Lock()
	defer m.Unlock()

	for _, opt := range opts {
		opt(&m.Opts)
	}

	r, ok := fromContext(m.Opts.Context)
	if !ok {
		r = make(map[string][]MockResponse)
	}
	m.Response = r

	return nil
}

func (m *MockClient) Options() client.Options {
	return m.Opts
}

func (m *MockClient) NewMessage(topic string, msg interface{}, opts ...client.MessageOption) client.Message {
	return m.Client.NewMessage(topic, msg, opts...)
}

func (m *MockClient) NewRequest(service, endpoint string, req interface{}, reqOpts ...client.RequestOption) client.Request {
	return m.Client.NewRequest(service, endpoint, req, reqOpts...)
}

func (m *MockClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	m.Lock()
	defer m.Unlock()

	response, ok := m.Response[req.Service()]
	if !ok {
		return errors.NotFound("go.micro.client.mock", "service not found")
	}

	for _, r := range response {
		if r.Endpoint != req.Endpoint() {
			continue
		}

		if r.Error != nil {
			return r.Error
		}

		v := reflect.ValueOf(rsp)

		if t := reflect.TypeOf(rsp); t.Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
		}
		response := r.Response
		if t := reflect.TypeOf(r.Response); t.Kind() == reflect.Func {
			var request []reflect.Value
			switch t.NumIn() {
			case 1:
				// one input params: (req)
				request = append(request, reflect.ValueOf(req.Body()))
			case 2:
				// two input params: (ctx, req)
				request = append(request, reflect.ValueOf(ctx), reflect.ValueOf(req.Body()))
			}

			responseValue := reflect.ValueOf(r.Response).Call(request)
			response = responseValue[0].Interface()
			if len(responseValue) == 2 {
				// make it possible to return error in response function
				respErr, ok := responseValue[1].Interface().(error)
				if ok && respErr != nil {
					return respErr
				}
			}
		}

		v.Set(reflect.ValueOf(response))

		return nil
	}

	return fmt.Errorf("rpc: can't find service %s", req.Endpoint())
}

func (m *MockClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	m.Lock()
	defer m.Unlock()

	// TODO: mock stream
	return nil, nil
}

func (m *MockClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	return nil
}

func (m *MockClient) String() string {
	return "mock"
}

func NewClient(opts ...client.Option) *MockClient {
	options := client.Options{
		Context: context.TODO(),
	}

	for _, opt := range opts {
		opt(&options)
	}

	r, ok := fromContext(options.Context)
	if !ok {
		r = make(map[string][]MockResponse)
	}

	return &MockClient{
		Client:   client.DefaultClient,
		Opts:     options,
		Response: r,
	}
}
