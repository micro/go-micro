package msgpackrpc

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

func TestRequest(t *testing.T) {
	r1 := Request{
		ID:     "100",
		Method: "Call",
		Body:   nil,
	}

	var buf bytes.Buffer

	if err := msgp.Encode(&buf, &r1); err != nil {
		t.Fatal(err)
	}

	var r2 Request

	if err := msgp.Decode(&buf, &r2); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r1, r2) {
		t.Error("values are not equal")
	}
}

func TestResponse(t *testing.T) {
	r1 := Response{
		ID:    "100",
		Error: "error",
	}

	var buf bytes.Buffer

	if err := msgp.Encode(&buf, &r1); err != nil {
		t.Fatal(err)
	}

	var r2 Response

	if err := msgp.Decode(&buf, &r2); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r1, r2) {
		t.Error("values are not equal")
	}
}

func TestNotification(t *testing.T) {
	r1 := Notification{
		Method: "Call",
		Body:   nil,
	}

	var buf bytes.Buffer

	if err := msgp.Encode(&buf, &r1); err != nil {
		t.Fatal(err)
	}

	var r2 Notification

	if err := msgp.Decode(&buf, &r2); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r1, r2) {
		t.Error("values are not equal")
	}
}

func BenchmarkRequestEncode(b *testing.B) {
	r := Request{
		ID:     "100",
		Method: "Call",
		Body:   nil,
	}

	var buf bytes.Buffer
	w := msgp.NewWriter(&buf)

	for i := 0; i < b.N; i++ {
		r.EncodeMsg(w)
		w.Flush()
		buf.Reset()
	}
}

func BenchmarkRequestDecode(b *testing.B) {
	r := Request{
		ID:     "100",
		Method: "Call",
		Body:   nil,
	}

	var buf bytes.Buffer
	msgp.Encode(&buf, &r)
	byts := buf.Bytes()

	mr := msgp.NewReader(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(byts)
		r.DecodeMsg(mr)
	}
}

func BenchmarkResponseEncode(b *testing.B) {
	r := Response{
		ID:    "100",
		Error: "error",
		Body:  nil,
	}

	var buf bytes.Buffer
	w := msgp.NewWriter(&buf)

	for i := 0; i < b.N; i++ {
		r.EncodeMsg(w)
		w.Flush()
		buf.Reset()
	}
}

func BenchmarkResponseDecode(b *testing.B) {
	r := Response{
		ID:    "100",
		Error: "error",
		Body:  nil,
	}

	var buf bytes.Buffer
	msgp.Encode(&buf, &r)
	byts := buf.Bytes()

	mr := msgp.NewReader(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(byts)
		r.DecodeMsg(mr)
	}
}

func BenchmarkNotificationEncode(b *testing.B) {
	r := Notification{
		Method: "Call",
		Body:   nil,
	}

	var buf bytes.Buffer
	w := msgp.NewWriter(&buf)

	for i := 0; i < b.N; i++ {
		r.EncodeMsg(w)
		w.Flush()
		buf.Reset()
	}
}

func BenchmarkNotificationDecode(b *testing.B) {
	r := Notification{
		Method: "Call",
		Body:   nil,
	}

	var buf bytes.Buffer
	msgp.Encode(&buf, &r)
	byts := buf.Bytes()

	mr := msgp.NewReader(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(byts)
		r.DecodeMsg(mr)
	}
}
