package client

import (
	"context"
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	results := []time.Duration{
		0 * time.Second,
		100 * time.Millisecond,
		600 * time.Millisecond,
		1900 * time.Millisecond,
		4300 * time.Millisecond,
		7900 * time.Millisecond,
	}

	c := NewClient()

	for i := 0; i < 5; i++ {
		d, err := exponentialBackoff(context.TODO(), c.NewRequest("test", "test", nil), i)
		if err != nil {
			t.Fatal(err)
		}

		if d != results[i] {
			t.Fatalf("Expected equal than %v, got %v", results[i], d)
		}
	}
}
