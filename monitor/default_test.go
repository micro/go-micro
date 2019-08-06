package monitor

import (
	"testing"
)

func TestMonitor(t *testing.T) {
	// create new monitor
	m := NewMonitor()

	if err := m.Run(); err != nil {
		t.Fatalf("failed to stop monitor: %v", err)
	}

	services := []string{"foo", "bar", "baz"}

	for _, service := range services {
		_, err := m.Status(service)
		if err == nil {
			t.Fatal("expected status error for unknown service")
		}

		if err := m.Watch(service); err == nil {
			t.Fatal("expected watch error for unknown service")
		}

		// TODO:
		// 1. start a service
		// 2. watch service
		// 3. get service status
	}

	// stop monitor
	if err := m.Stop(); err != nil {
		t.Fatalf("failed to stop monitor: %v", err)
	}
}
