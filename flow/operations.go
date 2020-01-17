package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/codec/bytes"
	pbFlow "github.com/micro/go-micro/flow/service/proto"
	"github.com/micro/go-micro/registry"
)

var (
	operations map[string]Operation
)

type Operations []Operation

func (ops Operations) String() string {
	rops := make([]string, 0, len(ops))
	for _, op := range ops {
		rops = append(rops, op.String())
	}

	return strings.Join(rops, ",")
}

func init() {
	operations = make(map[string]Operation)
	RegisterOperation(&sagaOperation{})
	RegisterOperation(&clientCallOperation{})
	RegisterOperation(&emptyOperation{})
	RegisterOperation(&aggregateOperation{})
	RegisterOperation(&modifyOperation{})
}

func RegisterOperation(op Operation) {
	if _, ok := operations[op.Type()]; ok {
		return
	}
	operations[op.Type()] = op
}

type sagaOperation struct {
	name    string
	forward Operation
	reverse Operation
	options OperationOptions
}

func SagaOperation(fwd Operation, rev Operation) *sagaOperation {
	return &sagaOperation{forward: fwd, reverse: rev, name: "saga_operation"}
}

func (op *sagaOperation) New() Operation {
	return &sagaOperation{}
}

func (op *sagaOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *sagaOperation) Name() string {
	return op.name
}

func (op *sagaOperation) Type() string {
	return "saga_operation"
}

func (op *sagaOperation) String() string {
	return op.name
}

func (op *sagaOperation) Encode() *pbFlow.Operation {
	return nil
}

func (op *sagaOperation) Decode(pb *pbFlow.Operation) {
}

func (op *sagaOperation) Options() OperationOptions {
	return op.options
}

func (op *sagaOperation) SetOptions(opts OperationOptions) {
	op.options = opts
}

type clientCallOperation struct {
	name     string
	service  string
	endpoint string
	options  OperationOptions
}

func ClientCallOperation(service, endpoint string, opts ...OperationOption) *clientCallOperation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &clientCallOperation{
		name:     fmt.Sprintf("%s.%s", service, endpoint),
		service:  service,
		endpoint: endpoint,
		options:  options,
	}
}

func (op *clientCallOperation) New() Operation {
	return &clientCallOperation{}
}

func (op *clientCallOperation) Execute(ctx context.Context, data []byte, opts ...ExecuteOption) ([]byte, error) {
	var err error

	options := ExecuteOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	req := client.NewRequest(op.service, op.endpoint, &bytes.Frame{Data: data})
	rsp := &bytes.Frame{}

	callOpts := []client.CallOption{}

	//moptions, ok := op.options.Context.Value()
	/*
		if len(op.options.SelectOptions) > 0 {
			callOpts = append(callOpts, client.WithSelectOption(op.options.SelectOptions...))
		}
		if len(op.options.Address) > 0 {
			callOpts = append(callOpts, client.WithAddress(op.options.Address...))
		}
		if len(op.options.CallWrappers) > 0 {
			callOpts = append(callOpts, client.WithCallWrapper(op.options.CallWrappers...))
		}
		if op.options.DialTimeout > 0 {
			callOpts = append(callOpts, client.WithDialTimeout(op.options.DialTimeout))
		}
		if op.options.RequestTimeout > 0 {
			callOpts = append(callOpts, client.WithRequestTimeout(op.options.RequestTimeout))
		}
	*/
	if err = op.options.Client.Call(ctx, req, rsp, callOpts...); err != nil {
		return nil, err
	}

	return rsp.Data, nil
}

func (op *clientCallOperation) Name() string {
	return op.name
}

func (op *clientCallOperation) Type() string {
	return "client_call_operation"
}

func (op *clientCallOperation) String() string {
	return op.name
}

func (op *clientCallOperation) Encode() *pbFlow.Operation {
	pb := &pbFlow.Operation{
		Name:    op.name,
		Type:    op.Type(),
		Options: make(map[string]string),
	}
	pb.Options["service"] = op.service
	pb.Options["endpoint"] = op.endpoint
	return pb
}

func (op *clientCallOperation) Decode(pb *pbFlow.Operation) {
	op.name = pb.Name
	op.service = pb.Options["service"]
	op.endpoint = pb.Options["endpoint"]
}

func (op *clientCallOperation) Options() OperationOptions {
	return op.options
}

func (op *clientCallOperation) SetOptions(opts OperationOptions) {
	op.options = opts
}

type emptyOperation struct {
	name    string
	options OperationOptions
}

func EmptyOperation(opts ...OperationOption) *emptyOperation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &emptyOperation{name: "empty_operation", options: options}
}

func (op *emptyOperation) New() Operation {
	return &emptyOperation{}
}

func (op *emptyOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *emptyOperation) Name() string {
	return op.name
}

func (op *emptyOperation) Type() string {
	return "empty_operation"
}

func (op *emptyOperation) String() string {
	return op.name
}

func (op *emptyOperation) Encode() *pbFlow.Operation {
	return nil
}

func (op *emptyOperation) Decode(pb *pbFlow.Operation) {
}

func (op *emptyOperation) Options() OperationOptions {
	return op.options
}

func (op *emptyOperation) SetOptions(opts OperationOptions) {
	op.options = opts
}

type aggregateOperation struct {
	name    string
	options OperationOptions
}

func AggregateOperation() *aggregateOperation {
	return &aggregateOperation{name: "aggregate_operation"}
}

func (op *aggregateOperation) New() Operation {
	return &aggregateOperation{}
}

func (op *aggregateOperation) Name() string {
	return op.name
}

func (op *aggregateOperation) Type() string {
	return "aggregate_operation"
}

func (op *aggregateOperation) String() string {
	return op.name
}

func (op *aggregateOperation) Encode() *pbFlow.Operation {
	return nil
}

func (op *aggregateOperation) Decode(pb *pbFlow.Operation) {
}

func (op *aggregateOperation) Execute(context.Context, []byte, ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *aggregateOperation) Options() OperationOptions {
	return op.options
}

func (op *aggregateOperation) SetOptions(opts OperationOptions) {
	op.options = opts
}

type modifyOperation struct {
	name    string
	options OperationOptions
	fn      func([]byte) ([]byte, error)
}

func ModifyOperation(fn func([]byte) ([]byte, error)) *modifyOperation {
	return &modifyOperation{name: "modify_operation"}
}

func (op *modifyOperation) New() Operation {
	return &modifyOperation{}
}

func (op *modifyOperation) Name() string {
	return op.name
}

func (op *modifyOperation) Type() string {
	return "modify_operation"
}

func (op *modifyOperation) String() string {
	return op.name
}

func (op *modifyOperation) Encode() *pbFlow.Operation {
	return nil
}

func (op *modifyOperation) Decode(pb *pbFlow.Operation) {
}

func (op *modifyOperation) Execute(context.Context, []byte, ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *modifyOperation) Options() OperationOptions {
	return op.options
}

func (op *modifyOperation) SetOptions(opts OperationOptions) {
	op.options = opts
}

type Operation interface {
	Name() string
	String() string
	Type() string
	New() Operation
	Decode(*pbFlow.Operation)
	Encode() *pbFlow.Operation
	Execute(context.Context, []byte, ...ExecuteOption) ([]byte, error)
	Options() OperationOptions
	SetOptions(OperationOptions)
}

type OperationOptions struct {
	Client    client.Client
	Broker    broker.Broker
	Registry  registry.Registry
	Timeout   time.Duration
	Retries   int
	AllowFail bool
	Context   context.Context
}

type OperationOption func(*OperationOptions)

func OperationTimeout(td time.Duration) OperationOption {
	return func(o *OperationOptions) {
		o.Timeout = td
	}
}

func OperationRetries(c int) OperationOption {
	return func(o *OperationOptions) {
		o.Retries = c
	}
}

func OperationAllowFail(b bool) OperationOption {
	return func(o *OperationOptions) {
		o.AllowFail = b
	}
}

func OperationContext(ctx context.Context) OperationOption {
	return func(o *OperationOptions) {
		o.Context = ctx
	}
}
