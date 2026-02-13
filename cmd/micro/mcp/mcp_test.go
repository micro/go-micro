package mcp

import (
	"reflect"
	"testing"
)

func TestParseTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     []string
	}{
		{
			name:     "simple two-part tool",
			toolName: "service.endpoint",
			want:     []string{"service", "endpoint"},
		},
		{
			name:     "three-part tool (service.Handler.Method)",
			toolName: "greeter.Greeter.Hello",
			want:     []string{"greeter", "Greeter", "Hello"},
		},
		{
			name:     "single part (invalid)",
			toolName: "service",
			want:     []string{"service"},
		},
		{
			name:     "four-part tool",
			toolName: "users.Users.Get.All",
			want:     []string{"users", "Users", "Get", "All"},
		},
		{
			name:     "empty string",
			toolName: "",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTool(tt.toolName)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTool(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}
