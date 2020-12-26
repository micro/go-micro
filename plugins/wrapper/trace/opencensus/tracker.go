package opencensus

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
)

type tracker struct {
	startedAt time.Time

	profile *StatsProfile
	span    *trace.Span

	method  string
	service string
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
		profile: profile,
		method:  req.Endpoint(),
		service: req.Service(),
	}
}

// newEventTracker creates a new tracker for a publication (client or server).
func newEventTracker(pub publicationDescriptor, profile *StatsProfile) *tracker {
	return &tracker{
		profile: profile,
		method:  pub.Topic(),
		service: "pubsub",
	}
}

// start monitoring a request. You can choose to let this method
// start a span for the request or attach one later.
func (t *tracker) start(ctx context.Context, startSpan bool) context.Context {
	t.startedAt = time.Now()

	ctx, _ = tag.New(ctx, tag.Upsert(Service, t.service), tag.Upsert(Endpoint, t.method))
	stats.Record(ctx, t.profile.CountMeasure.M(1))

	if startSpan {
		ctx, t.span = trace.StartSpan(
			ctx,
			fmt.Sprintf("rpc/%s/%s/%s", t.profile.Role, t.service, t.method),
		)
	}

	return ctx
}

// end a request's monitoring session. If there is a span ongoing, it will
// be ended and metrics will be recorded.
func (t *tracker) end(ctx context.Context, err error) {
	status := getResponseStatus(err)

	ctx, _ = tag.New(ctx, tag.Upsert(StatusCode, strconv.Itoa(int(status.Code))))
	stats.Record(ctx, t.profile.LatencyMeasure.M(float64(time.Since(t.startedAt))/float64(time.Millisecond)))

	if t.span != nil {
		t.span.SetStatus(status)
		t.span.End()
	}
}
