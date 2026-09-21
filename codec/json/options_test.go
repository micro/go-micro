package json

import (
	"bytes"
	stdjson "encoding/json"
	"go-micro.dev/v6/codec"
	"math"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func numericMessage(t *testing.T) *dynamicpb.Message {
	t.Helper()
	field := func(name string, n int32, kind descriptorpb.FieldDescriptorProto_Type, repeated bool) *descriptorpb.FieldDescriptorProto {
		label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
		if repeated {
			label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
		}
		return &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(n), Type: kind.Enum(), Label: label.Enum()}
	}
	f, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{Name: proto.String("numbers.proto"), Package: proto.String("numbers"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Sample"), Field: []*descriptorpb.FieldDescriptorProto{field("present_count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT64, false), field("active", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, false), field("amounts", 3, descriptorpb.FieldDescriptorProto_TYPE_UINT64, true), field("label", 4, descriptorpb.FieldDescriptorProto_TYPE_STRING, false)}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return dynamicpb.NewMessage(f.Messages().Get(0))
}

func TestProtoJSONOutputOptions(t *testing.T) {
	msg := numericMessage(t)
	options := Options{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}, Int64AsNumber: true}
	marshaler := Marshaler{Options: options}
	data, err := marshaler.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]interface{}
	if err = stdjson.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	if object["present_count"] != float64(0) || object["active"] != false || len(object["amounts"].([]interface{})) != 0 {
		t.Fatalf("zero values lost: %s", data)
	}
	fields := msg.Descriptor().Fields()
	msg.Set(fields.ByName("present_count"), protoreflect.ValueOfInt64(42))
	msg.Set(fields.ByName("label"), protoreflect.ValueOfString("42"))
	msg.Mutable(fields.ByName("amounts")).List().Append(protoreflect.ValueOfUint64(17))
	data, err = marshaler.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err = stdjson.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	if object["present_count"] != float64(42) || object["label"] != "42" || object["amounts"].([]interface{})[0] != float64(17) {
		t.Fatalf("wrong numeric conversion: %s", data)
	}
	decoded := dynamicpb.NewMessage(msg.Descriptor())
	if err = marshaler.Unmarshal(data, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(msg, decoded) {
		t.Fatal("round trip lost data")
	}
	msg.Set(fields.ByName("present_count"), protoreflect.ValueOfInt64(math.MaxInt64))
	if _, err = marshaler.Marshal(msg); err == nil {
		t.Fatal("unsafe integer accepted")
	}
	if _, err = (Marshaler{}).Marshal(msg); err != nil {
		t.Fatalf("default protobuf strings rejected: %v", err)
	}
}

func TestProtoJSONWrapperAndAnyIntegers(t *testing.T) {
	marshaler := Marshaler{Options: Options{Int64AsNumber: true}}
	for _, value := range []int64{-(1<<53 - 1), 1<<53 - 1} {
		msg := wrapperspb.Int64(value)
		data, err := marshaler.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 || data[0] == '"' {
			t.Fatalf("wrapper remains string: %s", data)
		}
		any, err := anypb.New(msg)
		if err != nil {
			t.Fatal(err)
		}
		data, err = marshaler.Marshal(any)
		if err != nil {
			t.Fatal(err)
		}
		var object map[string]interface{}
		if err = stdjson.Unmarshal(data, &object); err != nil {
			t.Fatal(err)
		}
		if _, ok := object["value"].(float64); !ok {
			t.Fatalf("Any wrapper remains string: %s", data)
		}
	}
	if _, err := marshaler.Marshal(wrapperspb.UInt64(math.MaxUint64)); err == nil {
		t.Fatal("unsafe unsigned integer accepted")
	}
	data, err := marshaler.Marshal(map[string]string{"amount": "123"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"amount":"123"}` {
		t.Fatalf("plain JSON changed: %s", data)
	}
}

type optionBuffer struct{ bytes.Buffer }

func (*optionBuffer) Close() error { return nil }

func TestConfiguredCodecRoundTrip(t *testing.T) {
	buffer := new(optionBuffer)
	c := NewCodecWithOptions(Options{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}, Int64AsNumber: true})(buffer)
	original := numericMessage(t)
	if err := c.Write(&codec.Message{}, original); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buffer.Bytes(), []byte(`"present_count":0`)) {
		t.Fatalf("configured codec omitted zero: %s", buffer.Bytes())
	}
	decoded := dynamicpb.NewMessage(original.Descriptor())
	if err := c.ReadBody(decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(original, decoded) {
		t.Fatal("codec round trip changed message")
	}
}
