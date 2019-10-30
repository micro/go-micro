package poller

import (
	"time"

	"github.com/micro/go-micro/runtime/build"
)

var (
	// DefaultPoller is default runtime poller
	DefaultPoller = NewHTTP(DefaultURL, DefaultPoll)
)

// Poller periodically poll for updates and returns the results
type Poller interface {
	// Poll polls for updates and returns results
	Poll() (*build.Build, error)
	// Tick returns poller tick time
	Tick() time.Duration
}
