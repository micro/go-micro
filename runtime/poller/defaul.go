package poller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/micro/go-micro/util/log"
)

var (
	// DefaultPoll defines how often we poll for updates
	DefaultPoll = 1 * time.Minute
	// DefaultURL defines url to poll for updates
	DefaultURL = "https://micro.m/update"
)

// Response is used to deserialise response returned by remote endpoint
type Response struct {
	// Commit is git commit sha
	Commit string `json:"commit,omitempty"`
	// Image is Docker build timestamp
	Image string `json:"image"`
	// Release is micro release tag
	Release string `json:"release,omitempty"`
}

// HTTP is http poller
type HTTP struct {
	// url to poll for updates
	url *url.URL
	// poll time to check for updates
	poll time.Duration
}

// NewHTTP creates HTTP poller and returns it
func NewHTTP(u string, poll time.Duration) (*HTTP, error) {
	// this should not return error, but lets make sure
	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	return &HTTP{
		url:  url,
		poll: poll,
	}, nil
}

// Poll polls for updates and returns results
func (h *HTTP) Poll() (string, error) {
	rsp, err := http.Get(h.url.String())
	if err != nil {
		log.Debugf("error polling updates: %v", err)
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != 200 {
		log.Debugf("error: unexpected http response: %v", rsp.StatusCode)
		return "", err
	}

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Debugf("error reading http response: %v", err)
		return "", err
	}

	// encoding format is assumed to be json
	var response *Response
	if err := json.Unmarshal(b, &response); err != nil {
		log.Debugf("error unmarshalling response: %v", err)
		return "", err
	}

	return response.Image, nil
}

// Tick returns poller tick time
func (h *HTTP) Tick() time.Duration {
	return h.poll
}
