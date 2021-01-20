package server

import (
	"fmt"
	"reflect"

	"github.com/asim/go-micro/v3/registry"
)

const (
	subSig = "func(context.Context, interface{}) error"
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
	opts       SubscriberOptions
}

func newSubscriber(topic string, sub interface{}, opts ...SubscriberOption) Subscriber {
	options := SubscriberOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&options)
	}

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
				"topic":      topic,
				"subscriber": "true",
			},
		})
	} else {
		hdlr := reflect.ValueOf(sub)
		name := reflect.Indirect(hdlr).Type().Name()

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
				Name:    name + "." + method.Name,
				Request: extractSubValue(method.Type),
				Metadata: map[string]string{
					"topic":      topic,
					"subscriber": "true",
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
		opts:       options,
	}
}

func validateSubscriber(sub Subscriber) error {
	typ := reflect.TypeOf(sub.Subscriber())
	var argType reflect.Type

	if typ.Kind() == reflect.Func {
		name := "Func"
		switch typ.NumIn() {
		case 2:
			argType = typ.In(1)
		default:
			return fmt.Errorf("subscriber %v takes wrong number of args: %v required signature %s", name, typ.NumIn(), subSig)
		}
		if !isExportedOrBuiltinType(argType) {
			return fmt.Errorf("subscriber %v argument type not exported: %v", name, argType)
		}
		if typ.NumOut() != 1 {
			return fmt.Errorf("subscriber %v has wrong number of outs: %v require signature %s",
				name, typ.NumOut(), subSig)
		}
		if returnType := typ.Out(0); returnType != typeOfError {
			return fmt.Errorf("subscriber %v returns %v not error", name, returnType.String())
		}
	} else {
		hdlr := reflect.ValueOf(sub.Subscriber())
		name := reflect.Indirect(hdlr).Type().Name()

		for m := 0; m < typ.NumMethod(); m++ {
			method := typ.Method(m)

			switch method.Type.NumIn() {
			case 3:
				argType = method.Type.In(2)
			default:
				return fmt.Errorf("subscriber %v.%v takes wrong number of args: %v required signature %s",
					name, method.Name, method.Type.NumIn(), subSig)
			}

			if !isExportedOrBuiltinType(argType) {
				return fmt.Errorf("%v argument type not exported: %v", name, argType)
			}
			if method.Type.NumOut() != 1 {
				return fmt.Errorf(
					"subscriber %v.%v has wrong number of outs: %v require signature %s",
					name, method.Name, method.Type.NumOut(), subSig)
			}
			if returnType := method.Type.Out(0); returnType != typeOfError {
				return fmt.Errorf("subscriber %v.%v returns %v not error", name, method.Name, returnType.String())
			}
		}
	}

	return nil
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

func (s *subscriber) Options() SubscriberOptions {
	return s.opts
}
