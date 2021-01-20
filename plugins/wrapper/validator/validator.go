package validator

import (
	"context"

	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/server"
)

type Validator interface {
	Validate() error
}

func NewHandlerWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if v, ok := req.Body().(Validator); ok {
				if err := v.Validate(); err != nil {
					return errors.BadRequest(req.Service(), "%v", err)
				}
			}
			return fn(ctx, req, rsp)
		}
	}
}
