package flow

import (
	"context"
	"fmt"
	"time"
)

type sagaOperation struct {
	name    string
	forward Operation
	reverse Operation
	options OperationOptions
}

func SagaOperation(fwd Operation, rev Operation) Operation {
	return &sagaOperation{forward: fwd, reverse: rev, name: "saga_operation"}
}

func (op *sagaOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *sagaOperation) Name() string {
	return op.name
}

func (op *sagaOperation) String() string {
	return op.name
}

func (op *sagaOperation) Encode() ([]byte, error) {
	return nil, nil
}

func (op *sagaOperation) Decode(data []byte) error {
	return nil
}

func (op *sagaOperation) Options() OperationOptions {
	return op.options
}

type clientCallOperation struct {
	name     string
	service  string
	endpoint string
	options  OperationOptions
}

func ClientCallOperation(service, endpoint string, opts ...OperationOption) Operation {
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

func (op *clientCallOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *clientCallOperation) Name() string {
	return op.name
}

func (op *clientCallOperation) String() string {
	return op.name
}

func (op *clientCallOperation) Encode() ([]byte, error) {
	return nil, nil
}

func (op *clientCallOperation) Decode(data []byte) error {
	return nil
}

func (op *clientCallOperation) Options() OperationOptions {
	return op.options
}

type emptyOperation struct {
	name    string
	options OperationOptions
}

func EmptyOperation(opts ...OperationOption) Operation {
	options := OperationOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &emptyOperation{name: "empty_operation", options: options}
}

func (op *emptyOperation) Execute(ctx context.Context, req []byte, opts ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *emptyOperation) Name() string {
	return op.name
}

func (op *emptyOperation) String() string {
	return op.name
}

func (op *emptyOperation) Encode() ([]byte, error) {
	return nil, nil
}

func (op *emptyOperation) Decode([]byte) error {
	return nil
}

func (op *emptyOperation) Options() OperationOptions {
	return op.options
}

type aggregateOperation struct {
	name    string
	options OperationOptions
}

func AggregateOperation() Operation {
	return &aggregateOperation{name: "aggregate_operation"}
}

func (op *aggregateOperation) Name() string {
	return op.name
}

func (op *aggregateOperation) String() string {
	return op.name
}

func (op *aggregateOperation) Encode() ([]byte, error) {
	return nil, nil
}

func (op *aggregateOperation) Decode([]byte) error {
	return nil
}

func (op *aggregateOperation) Execute(context.Context, []byte, ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *aggregateOperation) Options() OperationOptions {
	return op.options
}

type modifyOperation struct {
	name    string
	options OperationOptions
	fn      func([]byte) ([]byte, error)
}

func ModifyOperation(fn func([]byte) ([]byte, error)) Operation {
	return &modifyOperation{name: "modify_operation"}
}

func (op *modifyOperation) Name() string {
	return op.name
}

func (op *modifyOperation) String() string {
	return op.name
}

func (op *modifyOperation) Encode() ([]byte, error) {
	return nil, nil
}

func (op *modifyOperation) Decode([]byte) error {
	return nil
}

func (op *modifyOperation) Execute(context.Context, []byte, ...ExecuteOption) ([]byte, error) {
	return nil, nil
}

func (op *modifyOperation) Options() OperationOptions {
	return op.options
}

type Operation interface {
	Name() string
	String() string
	Decode([]byte) error
	Encode() ([]byte, error)
	Execute(context.Context, []byte, ...ExecuteOption) ([]byte, error)
	Options() OperationOptions
}

type OperationOptions struct {
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
