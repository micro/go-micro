package poller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/micro/go-micro/runtime/build"
	"github.com/micro/go-micro/util/log"
)

var (
	// DefaultPoll defines how often we poll for updates
	DefaultPoll = 1 * time.Minute
	// DefaultURL defines url to poll for updates
	DefaultURL = "https://micro.m/update"
)

// HTTP is http poller
type HTTP struct {
	// url to poll for updates
	url string
	// poll time to check for updates
	poll time.Duration
}

// NewHTTP creates HTTP poller and returns it
func NewHTTP(url string, poll time.Duration) *HTTP {
	return &HTTP{
		url:  url,
		poll: poll,
	}
}

// Poll polls for updates and returns results
func (h *HTTP) Poll() (*build.Build, error) {
	// this should not return error, but lets make sure
	url, err := url.Parse(h.url)
	if err != nil {
		return nil, err
	}

	rsp, err := http.Get(url.String())
	if err != nil {
		log.Debugf("error polling updates: %v", err)
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != 200 {
		log.Debugf("error: unexpected http response: %v", rsp.StatusCode)
		return nil, err
	}

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Debugf("error reading http response: %v", err)
		return nil, err
	}

	// encoding format is assumed to be json
	var build *build.Build
	if err := json.Unmarshal(b, &build); err != nil {
		log.Debugf("error unmarshalling response: %v", err)
		return nil, err
	}

	return build, nil
}

// Tick returns poller tick time
func (h *HTTP) Tick() time.Duration {
	return h.poll
}
