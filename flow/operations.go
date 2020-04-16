package flow

import (
	"context"
	"time"

	pbFlow "github.com/micro/go-micro/v2/flow/service/proto"
)

var (
	Operations map[string]Operation
)

func init() {
	Operations = make(map[string]Operation)
	RegisterOperation(&clientCallOperation{})
	RegisterOperation(&clientPublishOperation{})
	RegisterOperation(&flowExecuteOperation{})
	RegisterOperation(&emptyOperation{})
}

func RegisterOperation(op Operation) {
	if _, ok := Operations[op.Type()]; ok {
		return
	}
	Operations[op.Type()] = op
}

type Operation interface {
	Name() string
	String() string
	Type() string
	Clone() Operation
	Decode(*pbFlow.Operation)
	Encode() *pbFlow.Operation
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
