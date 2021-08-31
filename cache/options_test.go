package cache

import (
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	testData := map[string]struct {
		set        bool
		expiration time.Duration
		items      map[string]Item
	}{
		"DefaultOptions":  {false, DefaultExpiration, map[string]Item{}},
		"ModifiedOptions": {true, time.Second, map[string]Item{"test": {"hello go-micro", 0}}},
	}

	for k, d := range testData {
		t.Run(k, func(t *testing.T) {
			var opts Options

			if d.set {
				opts = NewOptions(
					Expiration(d.expiration),
					Items(d.items),
				)
			} else {
				opts = NewOptions()
			}

			// test options
			for _, o := range []Options{opts} {
				if o.Expiration != d.expiration {
					t.Fatalf("Expected expiration '%v', got '%v'", d.expiration, o.Expiration)
				}

				if o.Items["test"] != d.items["test"] {
					t.Fatalf("Expected items %#v, got %#v", d.items, o.Items)
				}
			}
		})
	}
}
