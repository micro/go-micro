package cache

import (
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	testData := map[string]struct {
		set        bool
		expiration time.Duration
	}{
		"DefaultOptions":  {false, DefaultExpiration},
		"ModifiedOptions": {true, time.Second},
	}

	for k, d := range testData {
		t.Run(k, func(t *testing.T) {
			var opts Options

			if d.set {
				opts = NewOptions(
					Expiration(d.expiration),
				)
			} else {
				opts = NewOptions()
			}

			// test options
			for _, o := range []Options{opts} {
				if o.Expiration != d.expiration {
					t.Fatalf("Expected expiration '%v', got '%v'", d.expiration, o.Expiration)
				}
			}
		})
	}
}
