package client

import (
	"testing"
)

func TestRequestOptions(t *testing.T) {
	r := newRequest("service", "method", nil, "application/json")
	if r.Service() != "service" {
		t.Fatalf("expected 'service' got %s", r.Service())
	}
	if r.Method() != "method" {
		t.Fatalf("expected 'method' got %s", r.Method())
	}
	if r.ContentType() != "application/json" {
		t.Fatalf("expected 'method' got %s", r.ContentType())
	}

	r2 := newRequest("service", "method", nil, "application/json", WithContentType("application/protobuf"))
	if r2.ContentType() != "application/protobuf" {
		t.Fatalf("expected 'method' got %s", r2.ContentType())
	}
}
