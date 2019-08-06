package monitor

import (
	"testing"
)

func TestMonitor(t *testing.T) {
	// create new monitor
	m := NewMonitor()

	services := []string{"foo", "bar", "baz"}

	for _, service := range services {
		_, err := m.Status(service)
		if err != nil {
			t.Fatal("expected status error for unknown service")
		}
	}
}
