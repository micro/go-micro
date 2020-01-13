// +build ignore

package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/codec/bytes"
)

type clientCallOp struct {
	node     string            `json:"node"`
	service  string            `json:"service"`
	endpoint string            `json:"endpoint"`
	options  ClientCallOptions `json:"options"`
}

type ClientCallOption func(*ClientCallOptions)

type ClientCallOptions struct {
	SelectOptions  []selector.SelectOption `json:"select_options"`
	Address        []string                `json:"address"`
	CallWrappers   []client.CallWrapper    `json:"call_wrappers"`
	DialTimeout    time.Duration           `json:"dial_timeout"`
	RequestTimeout time.Duration           `json:"request_timeout"`
}

func NewClientCall(node, service, endpoint string, opts ...ClientCallOption) *clientCallOp {
	op := &clientCallOp{
		node:     node,
		service:  service,
		endpoint: endpoint,
	}

	// parse options
	for _, o := range opts {
		o(&op.options)
	}

	return op
}

func WithAddress(a ...string) ClientCallOption {
	return func(o *ClientCallOptions) {
		o.Address = a
	}
}

func WithSelectOption(so ...selector.SelectOption) ClientCallOption {
	return func(o *ClientCallOptions) {
		o.SelectOptions = append(o.SelectOptions, so...)
	}
}

// WithRequestTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithRequestTimeout(td time.Duration) ClientCallOption {
	return func(o *ClientCallOptions) {
		o.RequestTimeout = td
	}
}

// WithDialTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithDialTimeout(td time.Duration) ClientCallOption {
	return func(o *ClientCallOptions) {
		o.DialTimeout = td
	}
}

// WithCallWrapper is a CallOption which adds to the existing CallFunc wrappers
func WithCallWrapper(cw ...client.CallWrapper) ClientCallOption {
	return func(o *ClientCallOptions) {
		o.CallWrappers = append(o.CallWrappers, cw...)
	}
}

func (s *clientCallOp) GetId() string {
	return fmt.Sprintf("go.micro.flow.call")
}

func (s *clientCallOp) GetProperties() map[string][]string {
	prop := make(map[string][]string)
	prop["node"] = []string{s.node}
	prop["service"] = []string{s.service}
	prop["endpoint"] = []string{s.endpoint}

	options, err := StructToMap(s.options, "json")
	if err != nil {
		panic(err)
		return nil
	}

	var opts []string
	for k, v := range options {
		opts = append(opts, fmt.Sprintf("%s=%v", k, v))
	}

	prop["options"] = opts

	return prop
}

func (s *clientCallOp) Decode(data []byte) error {
	err := json.Unmarshal(data, s)
	s.options = ClientCallOptions{}
	/*
		prop := opExp.Properties
		s.service = prop["service"][0]
		s.endpoint = prop["endpoint"][0]
		log.Printf("%#+v\n", prop)
	*/
	return err
}

func (s *clientCallOp) Execute(data []byte, options map[string]interface{}) ([]byte, error) {
	fmt.Printf("execute %#+v\n", s)
	switch s.service {
	//case "AccountCreate.Service", "MailerSend.Service":
	//	break
	default:
		n := rand.Intn(3)
		_ = n
		//	time.Sleep(time.Duration(n) * time.Second)
	}
	return []byte(fmt.Sprintf("%#+v\n", options)), nil

	req := client.NewRequest(s.service, s.endpoint, &bytes.Frame{Data: data})
	rsp := &bytes.Frame{}
	opts := []client.CallOption{}

	if len(s.options.SelectOptions) > 0 {
		opts = append(opts, client.WithSelectOption(s.options.SelectOptions...))
	}
	if len(s.options.Address) > 0 {
		opts = append(opts, client.WithAddress(s.options.Address...))
	}
	if len(s.options.CallWrappers) > 0 {
		opts = append(opts, client.WithCallWrapper(s.options.CallWrappers...))
	}
	if s.options.DialTimeout > 0 {
		opts = append(opts, client.WithDialTimeout(s.options.DialTimeout))
	}
	if s.options.RequestTimeout > 0 {
		opts = append(opts, client.WithRequestTimeout(s.options.RequestTimeout))
	}

	err := client.Call(context.Background(), req, rsp, opts...)
	if err != nil {
		return nil, err
	}
	return rsp.Data, nil
}

func (s *clientCallOp) Encode() []byte {
	buf, _ := json.Marshal(s)
	return buf
}
