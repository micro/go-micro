// Package cloudevents provides a cloudevents handler publishing the event using the go-micro/client
package cloudevents

import (
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/micro/go-micro/api/handler"
	"github.com/micro/go-micro/util/ctx"
)

type event struct {
	options handler.Options
}

var (
	Handler   = "cloudevents"
	versionRe = regexp.MustCompilePOSIX("^v[0-9]+$")
)

func eventName(parts []string) string {
	return strings.Join(parts, ".")
}

func evRoute(ns, p string) (string, string) {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")

	if len(p) == 0 {
		return ns, "event"
	}

	parts := strings.Split(p, "/")

	// no path
	if len(parts) == 0 {
		// topic: namespace
		// action: event
		return strings.Trim(ns, "."), "event"
	}

	// Treat /v[0-9]+ as versioning
	// /v1/foo/bar => topic: v1.foo action: bar
	if len(parts) >= 2 && versionRe.Match([]byte(parts[0])) {
		topic := ns + "." + strings.Join(parts[:2], ".")
		action := eventName(parts[1:])
		return topic, action
	}

	// /foo => topic: ns.foo action: foo
	// /foo/bar => topic: ns.foo action: bar
	topic := ns + "." + strings.Join(parts[:1], ".")
	action := eventName(parts[1:])

	return topic, action
}

func (e *event) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// request to topic:event
	// create event
	// publish to topic
	topic, _ := evRoute(e.options.Namespace, r.URL.Path)

	// create event
	ev, err := FromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// get client
	c := e.options.Service.Client()

	// create publication
	p := c.NewMessage(topic, ev)

	// publish event
	if err := c.Publish(ctx.FromRequest(r), p); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (e *event) String() string {
	return "cloudevents"
}

func NewHandler(opts ...handler.Option) handler.Handler {
	return &event{
		options: handler.NewOptions(opts...),
	}
}
