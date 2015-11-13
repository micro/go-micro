package server

import (
	"fmt"
	"reflect"

	"github.com/myodc/go-micro/registry"
)

func extractValue(v reflect.Type) *registry.Value {
	if v == nil {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	arg := &registry.Value{
		Name: v.Name(),
		Type: v.Name(),
	}

	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			val := extractValue(v.Field(i).Type)
			val.Name = v.Field(i).Name
			arg.Values = append(arg.Values, val)
		}
	}

	return arg
}

func extractEndpoint(method reflect.Method) *registry.Endpoint {
	if method.PkgPath != "" {
		return nil
	}

	var rspType, reqType reflect.Type
	var stream bool
	mt := method.Type

	switch mt.NumIn() {
	case 3:
		reqType = mt.In(1)
		rspType = mt.In(2)
	case 4:
		reqType = mt.In(2)
		rspType = mt.In(3)
	default:
		return nil
	}

	if rspType.Kind() == reflect.Func {
		stream = true
	}

	request := extractValue(reqType)
	response := extractValue(rspType)

	return &registry.Endpoint{
		Name:     method.Name,
		Request:  request,
		Response: response,
		Metadata: map[string]string{
			"stream": fmt.Sprintf("%v", stream),
		},
	}
}

func extractSubValue(typ reflect.Type) *registry.Value {
	var reqType reflect.Type
	switch typ.NumIn() {
	case 1:
		reqType = typ.In(0)
	case 2:
		reqType = typ.In(1)
	default:
		return nil
	}
	return extractValue(reqType)
}
