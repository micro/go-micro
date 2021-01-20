package datadog

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"

	microerr "github.com/asim/go-micro/v3/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type tracker struct {
	startedAt time.Time

	profile          *StatsProfile
	span             ddtrace.Span
	startSpanOptions []ddtrace.StartSpanOption

	reqEndpoint string
	reqService  string
}

type requestDescriptor interface {
	Service() string
	Endpoint() string
}

type publicationDescriptor interface {
	Topic() string
}

// newRequestTracker creates a new tracker for an RPC request (client or server).
func newRequestTracker(req requestDescriptor, profile *StatsProfile) *tracker {
	return &tracker{
		profile:     profile,
		reqService:  req.Service(),
		reqEndpoint: req.Endpoint(),
	}
}

// newEventTracker creates a new tracker for a publication (client or server).
func newEventTracker(pub publicationDescriptor, profile *StatsProfile) *tracker {
	return &tracker{
		profile:     profile,
		reqService:  "micro.pubsub",
		reqEndpoint: pub.Topic(),
	}
}

// start monitoring a request. You can choose to let this method
// start a span for the request or attach one later.
func (t *tracker) StartSpanFromContext(ctx context.Context) context.Context {
	t.startedAt = time.Now()

	opts := []ddtrace.StartSpanOption{
		tracer.ResourceName(t.reqEndpoint),
		tracer.SpanType(ext.AppTypeRPC),
		tracer.StartTime(t.startedAt),
	}

	t.span, ctx = StartSpanFromContext(ctx, t.profile.Role, opts...)

	return ctx
}

// finishWithError end a request's monitoring session. If there is a span ongoing, it will
// be ended.
func (t *tracker) finishWithError(err error, noDebugStack bool) {
	if t.span == nil {
		return
	}

	statusCode := codes.OK
	finishOptions := []tracer.FinishOption{
		tracer.FinishTime(time.Now()),
	}

	microErr, ok := err.(*microerr.Error)

	if ok {
		finishOptions = append(finishOptions, tracer.WithError(
			fmt.Errorf("%s: %s", microErr.Id, microErr.Detail),
		))

		c, ok := microCodeToStatusCode[microErr.Code]
		if ok {
			statusCode = c
		} else {
			statusCode = codes.Unknown
		}
	}

	t.span.SetTag(tagStatus, statusCode.String())

	if noDebugStack {
		finishOptions = append(finishOptions, tracer.NoDebugStack())
	}

	t.span.Finish(finishOptions...)
	t.span = nil
}
