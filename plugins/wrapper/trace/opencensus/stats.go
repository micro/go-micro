package opencensus

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// The following client RPC measures are supported for use in custom views.
var (
	ClientRequestCount = stats.Int64("opencensus.io/rpc/client/request_count", "Number of RPC requests started", stats.UnitNone)
	ClientLatency      = stats.Float64("opencensus.io/rpc/client/latency", "End-to-end latency", stats.UnitMilliseconds)
)

// The following server RPC measures are supported for use in custom views.
var (
	ServerRequestCount = stats.Int64("opencensus.io/rpc/server/request_count", "Number of RPC requests received", stats.UnitNone)
	ServerLatency      = stats.Float64("opencensus.io/rpc/server/latency", "End-to-end latency", stats.UnitMilliseconds)
)

// The following tags are applied to stats recorded by this package.
// Service and Method are applied to all measures.
// StatusCode is not applied to ClientRequestCount or ServerRequestCount,
// since it is recorded before the status is known.
var (
	// StatusCode is the RPC status code.
	StatusCode, _ = tag.NewKey("rpc.status")

	// Service is the name of the micro-service.
	Service, _ = tag.NewKey("rpc.service")

	// Method is the service method called.
	Endpoint, _ = tag.NewKey("rpc.endpoint")
)

// Default distributions used by views in this package.
var (
	DefaultLatencyDistribution = view.Distribution(0, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)
)

// This package provides some convenience views.
// You need to subscribe to the views for data to actually be collected.
var (
	ClientRequestCountView = &view.View{
		Name:        "opencensus.io/rpc/client/request_count",
		Description: "Count of RPC requests started",
		Measure:     ClientRequestCount,
		Aggregation: view.Count(),
	}

	ClientLatencyView = &view.View{
		Name:        "opencensus.io/rpc/client/latency",
		Description: "Latency distribution of RPC requests",
		Measure:     ClientLatency,
		Aggregation: DefaultLatencyDistribution,
	}

	ClientRequestCountByMethod = &view.View{
		Name:        "opencensus.io/rpc/client/request_count_by_method",
		Description: "Client request count by RPC method",
		TagKeys:     []tag.Key{Endpoint},
		Measure:     ClientRequestCount,
		Aggregation: view.Count(),
	}

	ClientResponseCountByStatusCode = &view.View{
		Name:        "opencensus.io/rpc/client/response_count_by_status_code",
		Description: "Client response count by RPC status code",
		TagKeys:     []tag.Key{StatusCode},
		Measure:     ClientLatency,
		Aggregation: view.Count(),
	}

	ServerRequestCountView = &view.View{
		Name:        "opencensus.io/rpc/server/request_count",
		Description: "Count of RPC requests received",
		Measure:     ServerRequestCount,
		Aggregation: view.Count(),
	}

	ServerLatencyView = &view.View{
		Name:        "opencensus.io/rpc/server/latency",
		Description: "Latency distribution of RPC requests",
		Measure:     ServerLatency,
		Aggregation: DefaultLatencyDistribution,
	}

	ServerRequestCountByMethod = &view.View{
		Name:        "opencensus.io/rpc/server/request_count_by_method",
		Description: "Server request count by RPC method",
		TagKeys:     []tag.Key{Endpoint},
		Measure:     ServerRequestCount,
		Aggregation: view.Count(),
	}

	ServerResponseCountByStatusCode = &view.View{
		Name:        "opencensus.io/rpc/server/response_count_by_status_code",
		Description: "Server response count by RPC status code",
		TagKeys:     []tag.Key{StatusCode},
		Measure:     ServerLatency,
		Aggregation: view.Count(),
	}
)

// DefaultClientViews are the default client views provided by this package.
var DefaultClientViews = []*view.View{
	ClientRequestCountView,
	ClientLatencyView,
	ClientRequestCountByMethod,
	ClientResponseCountByStatusCode,
}

// DefaultServerViews are the default server views provided by this package.
var DefaultServerViews = []*view.View{
	ServerRequestCountView,
	ServerLatencyView,
	ServerRequestCountByMethod,
	ServerResponseCountByStatusCode,
}

// StatsProfile groups metrics-related data.
type StatsProfile struct {
	Role           string
	CountMeasure   *stats.Int64Measure
	LatencyMeasure *stats.Float64Measure
}

var (
	// ClientProfile is used for RPC clients.
	ClientProfile = &StatsProfile{
		Role:           "client",
		CountMeasure:   ClientRequestCount,
		LatencyMeasure: ClientLatency,
	}

	// ServerProfile is used for RPC servers.
	ServerProfile = &StatsProfile{
		Role:           "server",
		CountMeasure:   ServerRequestCount,
		LatencyMeasure: ServerLatency,
	}
)
