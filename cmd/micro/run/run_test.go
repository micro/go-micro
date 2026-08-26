package run

import "testing"

func TestHasEnv(t *testing.T) {
	env := []string{"PATH=/bin", "OTEL_SERVICE_NAME=platform"}
	for _, tt := range []struct {
		key  string
		want bool
	}{
		{"OTEL_SERVICE_NAME", true},
		{"PATH", true},
		{"OTEL_EXPORTER_OTLP_ENDPOINT", false},
		{"OTEL", false}, // prefix alone must not match
	} {
		if got := hasEnv(env, tt.key); got != tt.want {
			t.Errorf("hasEnv(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
	if hasEnv(nil, "ANY") {
		t.Error("hasEnv(nil) = true, want false")
	}
}
