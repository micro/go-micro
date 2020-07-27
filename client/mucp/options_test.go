package mucp

import (
	"testing"
	"time"

	"github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/transport"
)

func TestCallOptions(t *testing.T) {
	testData := []struct {
		set      bool
		retries  int
		rtimeout time.Duration
		dtimeout time.Duration
	}{
		{false, client.DefaultRetries, client.DefaultRequestTimeout, transport.DefaultDialTimeout},
		{true, 10, time.Second, time.Second * 2},
	}

	for _, d := range testData {
		var opts client.Options
		var cl client.Client

		if d.set {
			opts = client.NewOptions(
				client.Retries(d.retries),
				client.RequestTimeout(d.rtimeout),
				client.DialTimeout(d.dtimeout),
			)

			cl = NewClient(
				client.Retries(d.retries),
				client.RequestTimeout(d.rtimeout),
				client.DialTimeout(d.dtimeout),
			)
		} else {
			opts = client.NewOptions()
			cl = NewClient()
		}

		// test options and those set in client
		for _, o := range []client.Options{opts, cl.Options()} {
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
			cretries := client.WithRetries(o.CallOptions.Retries * 10)
			crtimeout := client.WithRequestTimeout(o.CallOptions.RequestTimeout * (time.Second * 10))
			cdtimeout := client.WithDialTimeout(o.CallOptions.DialTimeout * (time.Second * 10))

			// set call options
			for _, opt := range []client.CallOption{cretries, crtimeout, cdtimeout} {
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
