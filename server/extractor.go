package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"go-micro.dev/v6/registry"
)

func extractValue(v reflect.Type, d int) *registry.Value {
	if d == 3 {
		return nil
	}
	if v == nil {
		return nil
	}

	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	arg := &registry.Value{
		Name: v.Name(),
		Type: v.Name(),
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.PkgPath != "" {
				continue
			}
			val := extractValue(f.Type, d+1)
			if val == nil {
				continue
			}

			// if we can find a json tag use it
			if tags := f.Tag.Get("json"); len(tags) > 0 {
				parts := strings.Split(tags, ",")
				if parts[0] == "-" || parts[0] == "omitempty" {
					continue
				}
				val.Name = parts[0]
			} else {
				val.Name = f.Name
			}

			// if there's no name default it
			if len(val.Name) == 0 {
				val.Name = v.Field(i).Name
			}

			// still no name then continue
			if len(val.Name) == 0 {
				continue
			}

			arg.Values = append(arg.Values, val)
		}
	case reflect.Slice:
		p := v.Elem()
		if p.Kind() == reflect.Pointer {
			p = p.Elem()
		}
		arg.Type = "[]" + p.Name()
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

	// are we dealing with a stream?
	switch rspType.Kind() {
	case reflect.Func, reflect.Interface:
		stream = true
	}

	request := extractValue(reqType, 0)
	response := extractValue(rspType, 0)

	ep := &registry.Endpoint{
		Name:     method.Name,
		Request:  request,
		Response: response,
		Metadata: make(map[string]string),
	}

	// set endpoint metadata for stream
	if stream {
		ep.Metadata = map[string]string{
			"stream": fmt.Sprintf("%v", stream),
		}
	}

	// Carry per-field descriptions from the request struct's `description`
	// struct tags so the MCP/API gateways can emit meaningful input schemas
	// without re-reading handler source at runtime.
	if descs := fieldDescriptions(reqType); len(descs) > 0 {
		if b, err := json.Marshal(descs); err == nil {
			ep.Metadata["request_fields"] = string(b)
		}
	}

	return ep
}

// fieldDescriptions maps each exportable top-level request field (by JSON
// name) to its `description` struct tag, if set.
func fieldDescriptions(typ reflect.Type) map[string]string {
	if typ == nil {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}
	var out map[string]string
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		desc := f.Tag.Get("description")
		if desc == "" {
			continue
		}
		name := f.Name
		if tags := f.Tag.Get("json"); len(tags) > 0 {
			parts := strings.Split(tags, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}
		if out == nil {
			out = make(map[string]string)
		}
		out[name] = desc
	}
	return out
}

func extractSubValue(typ reflect.Type) *registry.Value {
	var reqType reflect.Type
	switch typ.NumIn() {
	case 1:
		reqType = typ.In(0)
	case 2:
		reqType = typ.In(1)
	case 3:
		reqType = typ.In(2)
	default:
		return nil
	}
	return extractValue(reqType, 0)
}
