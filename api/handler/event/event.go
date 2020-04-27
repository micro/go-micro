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
	"github.com/micro/go-micro/v2/api/handler"
	proto "github.com/micro/go-micro/v2/api/proto"
	"github.com/micro/go-micro/v2/util/ctx"
	"github.com/oxtoacart/bpool"
)

var (
	bufferPool = bpool.NewSizedBufferPool(1024, 8)
)

type event struct {
	opts handler.Options
}

var (
	Handler   = "event"
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
	bsize := handler.DefaultMaxRecvSize
	if e.opts.MaxRecvSize > 0 {
		bsize = e.opts.MaxRecvSize
	}

	r.Body = http.MaxBytesReader(w, r.Body, bsize)

	// request to topic:event
	// create event
	// publish to topic

	topic, action := evRoute(e.opts.Namespace, r.URL.Path)

	// create event
	ev := &proto.Event{
		Name: action,
		// TODO: dedupe event
		Id:        fmt.Sprintf("%s-%s-%s", topic, action, uuid.New().String()),
		Header:    make(map[string]*proto.Pair),
		Timestamp: time.Now().Unix(),
	}

	// set headers
	for key, vals := range r.Header {
		header, ok := ev.Header[key]
		if !ok {
			header = &proto.Pair{
				Key: key,
			}
			ev.Header[key] = header
		}
		header.Values = vals
	}

	// set body
	if r.Method == "GET" {
		bytes, _ := json.Marshal(r.URL.Query())
		ev.Data = string(bytes)
	} else {
		// Read body
		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		if _, err := buf.ReadFrom(r.Body); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		ev.Data = buf.String()
	}

	// get client
	c := e.opts.Client

	// create publication
	p := c.NewMessage(topic, ev)

	// publish event
	if err := c.Publish(ctx.FromRequest(r), p); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (e *event) String() string {
	return "event"
}

func NewHandler(opts ...handler.Option) handler.Handler {
	return &event{
		opts: handler.NewOptions(opts...),
	}
}
