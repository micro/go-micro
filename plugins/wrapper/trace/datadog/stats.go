package datadog

// TODO: add statd
// https://github.com/DataDog/datadog-go/tree/master/statsd

// StatsProfile groups metrics-related data.
type StatsProfile struct {
	Role string
}

var (
	// ClientProfile is used for RPC clients.
	ClientProfile = &StatsProfile{
		Role: "micro.client",
	}

	// ServerProfile is used for RPC servers.
	ServerProfile = &StatsProfile{
		Role: "micro.server",
	}
)
