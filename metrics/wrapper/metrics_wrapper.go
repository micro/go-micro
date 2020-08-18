package wrapper

import (
	"time"

	"context"

	"github.com/micro/go-micro/v3/metrics"
	"github.com/micro/go-micro/v3/server"
)

// Wrapper provides a HandlerFunc for metrics.Reporter implementations:
type Wrapper struct {
	reporter metrics.Reporter
}

// New returns a *Wrapper configured with the given metrics.Reporter:
func New(reporter metrics.Reporter) *Wrapper {
	return &Wrapper{
		reporter: reporter,
	}
}

// HandlerFunc instruments handlers registered to a service:
func (w *Wrapper) HandlerFunc(handlerFunction server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {

		// Build some tags to describe the call:
		tags := metrics.Tags{
			"method": req.Method(),
		}

		// Start the clock:
		callTime := time.Now()

		// Run the handlerFunction:
		err := handlerFunction(ctx, req, rsp)

		// Add a result tag:
		if err != nil {
			tags["result"] = "failure"
		} else {
			tags["result"] = "failure"
		}

		// Instrument the result (if the DefaultClient has been configured):
		w.reporter.Timing("service.handler", time.Since(callTime), tags)

		return err
	}
}
