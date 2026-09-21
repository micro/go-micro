package json

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Options configures protobuf JSON without changing plain Go JSON encoding.
// The zero value preserves the existing protobuf JSON wire format.
type Options struct {
	MarshalOptions   protojson.MarshalOptions
	UnmarshalOptions protojson.UnmarshalOptions
	// Int64AsNumber emits protobuf 64-bit integer values as JSON numbers only
	// within JavaScript's exact integer range, rejecting larger values. Map keys
	// remain strings. By default protobuf JSON encodes 64-bit integers as strings.
	Int64AsNumber bool
}

func (o Options) numericIntegers(data []byte, descriptor protoreflect.MessageDescriptor) ([]byte, error) {
	var value interface{}
	decoder := stdjson.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	value, err := o.convertMessage(value, descriptor)
	if err != nil {
		return nil, err
	}
	return stdjson.Marshal(value)
}

func (o Options) convertMessage(value interface{}, descriptor protoreflect.MessageDescriptor) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch descriptor.FullName() {
	case "google.protobuf.Int64Value":
		return exactInteger(value, true)
	case "google.protobuf.UInt64Value":
		return exactInteger(value, false)
	case "google.protobuf.Any":
		object, ok := value.(map[string]interface{})
		if !ok {
			return value, nil
		}
		url, _ := object["@type"].(string)
		resolver := o.MarshalOptions.Resolver
		if resolver == nil {
			resolver = protoregistry.GlobalTypes
		}
		message, err := resolver.FindMessageByURL(url)
		if err != nil {
			return nil, err
		}
		desc := message.Descriptor()
		if inner, ok := object["value"]; ok && isWellKnown(desc.FullName()) {
			converted, err := o.convertMessage(inner, desc)
			if err != nil {
				return nil, err
			}
			object["value"] = converted
			return object, nil
		}
		return o.convertMessage(object, desc)
	}
	object, ok := value.(map[string]interface{})
	if !ok {
		return value, nil
	}
	fields := descriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		key := field.JSONName()
		if o.MarshalOptions.UseProtoNames {
			key = string(field.Name())
		}
		current, exists := object[key]
		if !exists || current == nil {
			continue
		}
		var converted interface{}
		var err error
		switch {
		case field.IsMap():
			values, ok := current.(map[string]interface{})
			if !ok {
				continue
			}
			for k, v := range values {
				values[k], err = o.convertField(v, field.MapValue())
				if err != nil {
					break
				}
			}
			converted = values
		case field.IsList():
			values, ok := current.([]interface{})
			if !ok {
				continue
			}
			for j, v := range values {
				values[j], err = o.convertField(v, field)
				if err != nil {
					break
				}
			}
			converted = values
		default:
			converted, err = o.convertField(current, field)
		}
		if err != nil {
			return nil, fmt.Errorf("protobuf JSON field %s: %w", field.FullName(), err)
		}
		object[key] = converted
	}
	return object, nil
}

func isWellKnown(name protoreflect.FullName) bool {
	switch name {
	case "google.protobuf.Any", "google.protobuf.Timestamp", "google.protobuf.Duration", "google.protobuf.FieldMask", "google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue", "google.protobuf.DoubleValue", "google.protobuf.FloatValue", "google.protobuf.Int64Value", "google.protobuf.UInt64Value", "google.protobuf.Int32Value", "google.protobuf.UInt32Value", "google.protobuf.BoolValue", "google.protobuf.StringValue", "google.protobuf.BytesValue":
		return true
	}
	return false
}

func (o Options) convertField(value interface{}, field protoreflect.FieldDescriptor) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch field.Kind() {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return exactInteger(value, true)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return exactInteger(value, false)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return o.convertMessage(value, field.Message())
	}
	return value, nil
}

func exactInteger(value interface{}, signed bool) (interface{}, error) {
	text, ok := value.(string)
	if !ok {
		return value, nil
	}
	const maximum = 1<<53 - 1
	if signed {
		number, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, err
		}
		if number < -maximum || number > maximum {
			return nil, fmt.Errorf("integer %s exceeds exact JSON number range", text)
		}
	} else {
		number, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			return nil, err
		}
		if number > maximum {
			return nil, fmt.Errorf("integer %s exceeds exact JSON number range", text)
		}
	}
	return stdjson.Number(text), nil
}
