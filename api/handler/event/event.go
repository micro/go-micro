// Package event provides a handler which publishes an event
package event

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oxtoacart/bpool"
	"go-micro.dev/v4/api/handler"
	proto "go-micro.dev/v4/api/proto"
	"go-micro.dev/v4/util/ctx"
)

var (
	bufferPool = bpool.NewSizedBufferPool(1024, 8)
)

type event struct {
	opts handler.Options
}

var (
	// Handler is the name of this handler.
	Handler   = "event"
	versionRe = regexp.MustCompilePOSIX("^v[0-9]+$")
)

func eventName(parts []string) string {
	return strings.Join(parts, ".")
}

func evRoute(namespace, myPath string) (string, string) {
	myPath = path.Clean(myPath)
	myPath = strings.TrimPrefix(myPath, "/")

	if len(myPath) == 0 {
		return namespace, Handler
	}

	parts := strings.Split(myPath, "/")

	// no path
	if len(parts) == 0 {
		// topic: namespace
		// action: event
		return strings.Trim(namespace, "."), Handler
	}

	// Treat /v[0-9]+ as versioning
	// /v1/foo/bar => topic: v1.foo action: bar
	if len(parts) >= 2 && versionRe.Match([]byte(parts[0])) {
		topic := namespace + "." + strings.Join(parts[:2], ".")
		action := eventName(parts[1:])

		return topic, action
	}

	// /foo => topic: ns.foo action: foo
	// /foo/bar => topic: ns.foo action: bar
	topic := namespace + "." + strings.Join(parts[:1], ".")
	action := eventName(parts[1:])

	return topic, action
}

func (e *event) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	bsize := handler.DefaultMaxRecvSize
	if e.opts.MaxRecvSize > 0 {
		bsize = e.opts.MaxRecvSize
	}

	req.Body = http.MaxBytesReader(rsp, req.Body, bsize)

	// request to topic:event
	// create event
	// publish to topic

	topic, action := evRoute(e.opts.Namespace, req.URL.Path)

	// create event
	event := &proto.Event{
		Name: action,
		// TODO: dedupe event
		Id:        fmt.Sprintf("%s-%s-%s", topic, action, uuid.New().String()),
		Header:    make(map[string]*proto.Pair),
		Timestamp: time.Now().Unix(),
	}

	// set headers
	for key, vals := range req.Header {
		header, ok := event.Header[key]
		if !ok {
			header = &proto.Pair{
				Key: key,
			}
			event.Header[key] = header
		}

		header.Values = vals
	}

	// set body
	if req.Method == http.MethodGet {
		bytes, _ := json.Marshal(req.URL.Query())
		event.Data = string(bytes)
	} else {
		// Read body
		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		if _, err := buf.ReadFrom(req.Body); err != nil {
			http.Error(rsp, err.Error(), http.StatusInternalServerError)

			return
		}
		event.Data = buf.String()
	}

	// get client
	c := e.opts.Client

	// create publication
	p := c.NewMessage(topic, event)

	// publish event
	if err := c.Publish(ctx.FromRequest(req), p); err != nil {
		http.Error(rsp, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (e *event) String() string {
	return Handler
}

// NewHandler returns a new event handler.
func NewHandler(opts ...handler.Option) handler.Handler {
	return &event{
		opts: handler.NewOptions(opts...),
	}
}
