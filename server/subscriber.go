package server

import (
	"encoding/json"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/kynrai/go-micro/broker"
	c "github.com/kynrai/go-micro/context"
	"github.com/kynrai/go-micro/registry"
	"golang.org/x/net/context"
)

type handler struct {
	method  reflect.Value
	reqType reflect.Type
	ctxType reflect.Type
}

type subscriber struct {
	topic      string
	rcvr       reflect.Value
	typ        reflect.Type
	subscriber interface{}
	handlers   []*handler
	endpoints  []*registry.Endpoint
}

func newSubscriber(topic string, sub interface{}) Subscriber {
	var endpoints []*registry.Endpoint
	var handlers []*handler

	if typ := reflect.TypeOf(sub); typ.Kind() == reflect.Func {
		h := &handler{
			method: reflect.ValueOf(sub),
		}

		switch typ.NumIn() {
		case 1:
			h.reqType = typ.In(0)
		case 2:
			h.ctxType = typ.In(0)
			h.reqType = typ.In(1)
		}

		handlers = append(handlers, h)

		endpoints = append(endpoints, &registry.Endpoint{
			Name:    "Func",
			Request: extractSubValue(typ),
			Metadata: map[string]string{
				"topic": topic,
			},
		})
	} else {
		for m := 0; m < typ.NumMethod(); m++ {
			method := typ.Method(m)
			h := &handler{
				method: method.Func,
			}

			switch method.Type.NumIn() {
			case 2:
				h.reqType = method.Type.In(1)
			case 3:
				h.ctxType = method.Type.In(1)
				h.reqType = method.Type.In(2)
			}

			handlers = append(handlers, h)

			endpoints = append(endpoints, &registry.Endpoint{
				Name:    method.Name,
				Request: extractSubValue(method.Type),
				Metadata: map[string]string{
					"topic": topic,
				},
			})
		}
	}

	return &subscriber{
		rcvr:       reflect.ValueOf(sub),
		typ:        reflect.TypeOf(sub),
		topic:      topic,
		subscriber: sub,
		handlers:   handlers,
		endpoints:  endpoints,
	}
}

func createSubHandler(sb *subscriber) broker.Handler {
	return func(msg *broker.Message) {
		hdr := make(map[string]string)
		for k, v := range msg.Header {
			hdr[k] = v
		}
		delete(hdr, "Content-Type")
		ctx := c.WithMetadata(context.Background(), hdr)
		rctx := reflect.ValueOf(ctx)

		for _, handler := range sb.handlers {
			var isVal bool
			var req reflect.Value
			var uerr error

			if handler.reqType.Kind() == reflect.Ptr {
				req = reflect.New(handler.reqType.Elem())
			} else {
				req = reflect.New(handler.reqType)
				isVal = true
			}

			switch msg.Header["Content-Type"] {
			case "application/octet-stream":
				uerr = proto.Unmarshal(msg.Body, req.Interface().(proto.Message))
			case "application/json":
				uerr = json.Unmarshal(msg.Body, req.Interface())
			}

			if uerr != nil {
				continue
			}

			if isVal {
				req = req.Elem()
			}

			var vals []reflect.Value
			if sb.typ.Kind() != reflect.Func {
				vals = append(vals, sb.rcvr)
			}

			if handler.ctxType != nil {
				vals = append(vals, rctx)
			}

			vals = append(vals, req)
			go handler.method.Call(vals)
		}
	}
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Subscriber() interface{} {
	return s.subscriber
}

func (s *subscriber) Endpoints() []*registry.Endpoint {
	return s.endpoints
}
