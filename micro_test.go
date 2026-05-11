package micro

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	svc := New("test-service")
	if svc == nil {
		t.Fatal("New returned nil")
	}
	if svc.Name() != "test-service" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "test-service")
	}
}

func TestNewWithAddress(t *testing.T) {
	svc := New("test-service", Address(":0"))
	if svc == nil {
		t.Fatal("New returned nil")
	}
	if svc.Name() != "test-service" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "test-service")
	}
}

func TestNewGroup(t *testing.T) {
	svc1 := New("svc1", Address(":0"))
	svc2 := New("svc2", Address(":0"))
	g := NewGroup(svc1, svc2)
	if g == nil {
		t.Fatal("NewGroup returned nil")
	}
}

func TestNewContext(t *testing.T) {
	svc := New("ctx-test")
	ctx := NewContext(context.Background(), svc)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext returned false")
	}
	if got.Name() != "ctx-test" {
		t.Errorf("FromContext Name() = %q, want %q", got.Name(), "ctx-test")
	}
}

func TestFromContextEmpty(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("FromContext on empty context should return false")
	}
}

func TestNewEvent(t *testing.T) {
	ev := NewEvent("test.topic", nil)
	if ev == nil {
		t.Fatal("NewEvent returned nil")
	}
}

func TestRegisterHandler(t *testing.T) {
	svc := New("handler-test", Address(":0"))
	type Handler struct{}
	err := RegisterHandler(svc.Server(), &Handler{})
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}
}
