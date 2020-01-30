package client

import (
	"testing"
	"time"

	"github.com/micro/go-micro/v2/transport"
)

func TestCallOptions(t *testing.T) {
	testData := []struct {
		set      bool
		retries  int
		rtimeout time.Duration
		dtimeout time.Duration
	}{
		{false, DefaultRetries, DefaultRequestTimeout, transport.DefaultDialTimeout},
		{true, 10, time.Second, time.Second * 2},
	}

	for _, d := range testData {
		var opts Options
		var cl Client

		if d.set {
			opts = NewOptions(
				Retries(d.retries),
				RequestTimeout(d.rtimeout),
				DialTimeout(d.dtimeout),
			)

			cl = NewClient(
				Retries(d.retries),
				RequestTimeout(d.rtimeout),
				DialTimeout(d.dtimeout),
			)
		} else {
			opts = NewOptions()
			cl = NewClient()
		}

		// test options and those set in client
		for _, o := range []Options{opts, cl.Options()} {
			if o.CallOptions.Retries != d.retries {
				t.Fatalf("Expected retries %v got %v", d.retries, o.CallOptions.Retries)
			}

			if o.CallOptions.RequestTimeout != d.rtimeout {
				t.Fatalf("Expected request timeout %v got %v", d.rtimeout, o.CallOptions.RequestTimeout)
			}

			if o.CallOptions.DialTimeout != d.dtimeout {
				t.Fatalf("Expected %v got %v", d.dtimeout, o.CallOptions.DialTimeout)
			}

			// copy CallOptions
			callOpts := o.CallOptions

			// create new opts
			cretries := WithRetries(o.CallOptions.Retries * 10)
			crtimeout := WithRequestTimeout(o.CallOptions.RequestTimeout * (time.Second * 10))
			cdtimeout := WithDialTimeout(o.CallOptions.DialTimeout * (time.Second * 10))

			// set call options
			for _, opt := range []CallOption{cretries, crtimeout, cdtimeout} {
				opt(&callOpts)
			}

			// check call options
			if e := o.CallOptions.Retries * 10; callOpts.Retries != e {
				t.Fatalf("Expected retries %v got %v", e, callOpts.Retries)
			}

			if e := o.CallOptions.RequestTimeout * (time.Second * 10); callOpts.RequestTimeout != e {
				t.Fatalf("Expected request timeout %v got %v", e, callOpts.RequestTimeout)
			}

			if e := o.CallOptions.DialTimeout * (time.Second * 10); callOpts.DialTimeout != e {
				t.Fatalf("Expected %v got %v", e, callOpts.DialTimeout)
			}

		}

	}
}
