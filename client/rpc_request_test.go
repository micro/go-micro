package client

import (
	"testing"
)

func TestRequestOptions(t *testing.T) {
	r := newRequest("service", "endpoint", nil, "application/json")
	if r.Service() != "service" {
		t.Fatalf("expected 'service' got %s", r.Service())
	}
	if r.Endpoint() != "endpoint" {
		t.Fatalf("expected 'endpoint' got %s", r.Endpoint())
	}
	if r.ContentType() != "application/json" {
		t.Fatalf("expected 'endpoint' got %s", r.ContentType())
	}

	r2 := newRequest("service", "endpoint", nil, "application/json", WithContentType("application/protobuf"))
	if r2.ContentType() != "application/protobuf" {
		t.Fatalf("expected 'endpoint' got %s", r2.ContentType())
	}
}
