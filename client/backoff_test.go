package client

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	delta := time.Duration(0)

	for i := 0; i < 5; i++ {
		d, err := exponentialBackoff(context.TODO(), NewJsonRequest("test", "test", nil), i)
		if err != nil {
			t.Fatal(err)
		}

		if d < delta {
			t.Fatalf("Expected greater than %v, got %v", delta, d)
		}

		delta = time.Millisecond * time.Duration(math.Pow(10, float64(i+1)))
	}
}
