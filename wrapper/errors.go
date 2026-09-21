// Package wrapper provides optional RPC middleware.
package wrapper

import (
	"context"

	"go-micro.dev/v6/errors"
	"go-micro.dev/v6/logger"
	"go-micro.dev/v6/server"
)

// LogRawErrors logs handler failures before transport error translation and
// returns them unchanged. Install explicitly where raw error details may be
// written to the configured log sink. Request payloads are not recorded.
// It cannot recover details a handler already replaced with a masked error.
func LogRawErrors(log logger.Logger) server.HandlerWrapper {
	log = logger.LoggerOrDefault(log)
	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			err := next(ctx, req, rsp)
			if err != nil {
				_, structured := errors.As(err)
				log.Fields(map[string]interface{}{"service": req.Service(), "endpoint": req.Endpoint(), "structured_error": structured}).Logf(logger.ErrorLevel, "RPC handler failed: %v", err)
			}
			return err
		}
	}
}
