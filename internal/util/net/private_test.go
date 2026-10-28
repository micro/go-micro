package net_test

import (
	"testing"

	mnet "go-micro.dev/v6/internal/util/net"
)

func TestIsPrivateAddress(t *testing.T) {
	tests := []struct {
		address string
		want    bool
	}{
		{"localhost:8080", true},
		{"127.0.0.1:8080", true},
		{"[::1]:8080", true},
		{"[::]:8080", true},
		{"10.0.0.8:8080", true},
		{"172.20.0.4:8080", true},
		{"192.168.1.2:8080", true},
		{"169.254.1.1:8080", true},
		{"8.8.8.8:443", false},
		{"services.example.com:443", false},
	}
	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			if got := mnet.PrivateAddress(tt.address); got != tt.want {
				t.Fatalf("PrivateAddress(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}
