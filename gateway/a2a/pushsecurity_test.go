package a2a

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDefaultPushURLPolicy(t *testing.T) {
	// Resolve test hostnames deterministically without real DNS.
	orig := pushLookupIP
	pushLookupIP = func(host string) ([]net.IP, error) {
		switch host {
		case "internal.example":
			return []net.IP{net.ParseIP("10.1.2.3")}, nil
		case "public.example":
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		case "rebind.example":
			// A host that resolves to both a public and an internal IP must be
			// rejected — any blocked address is disqualifying.
			return []net.IP{net.ParseIP("93.184.216.34"), net.ParseIP("127.0.0.1")}, nil
		}
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	defer func() { pushLookupIP = orig }()

	blocked := []string{
		"http://127.0.0.1/hook",              // loopback
		"http://169.254.169.254/latest/meta", // cloud metadata (link-local)
		"http://10.0.0.5/hook",               // RFC1918
		"http://[::1]/hook",                  // IPv6 loopback
		"http://[fd00::1]/hook",              // IPv6 ULA (private)
		"http://0.0.0.0/hook",                // unspecified
		"http://internal.example/hook",       // hostname → private
		"http://rebind.example/hook",         // one internal IP among many
		"ftp://public.example/hook",          // non-http(s) scheme
		"file:///etc/passwd",                 // scheme
		"http:///nohost",                     // no host
	}
	for _, raw := range blocked {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if err := defaultPushURLPolicy(u); err == nil {
			t.Errorf("defaultPushURLPolicy(%q) = nil, want blocked", raw)
		}
	}

	allowed := []string{
		"http://93.184.216.34/hook",   // public literal IP
		"https://public.example/hook", // hostname → public
	}
	for _, raw := range allowed {
		u, _ := url.Parse(raw)
		if err := defaultPushURLPolicy(u); err != nil {
			t.Errorf("defaultPushURLPolicy(%q) = %v, want allowed", raw, err)
		}
	}
}

func TestPushDialControlBlocksPrivate(t *testing.T) {
	blocked := []string{"127.0.0.1:80", "169.254.169.254:80", "10.0.0.1:443", "[::1]:80", "0.0.0.0:80"}
	for _, addr := range blocked {
		if err := pushDialControl("tcp", addr, nil); err == nil {
			t.Errorf("pushDialControl(%q) = nil, want blocked", addr)
		}
	}
	if err := pushDialControl("tcp", "8.8.8.8:443", nil); err != nil {
		t.Errorf("pushDialControl(public) = %v, want allowed", err)
	}
}

// TestSetPushConfigRejectsSSRFURL: an untrusted caller cannot register a
// callback pointing at an internal address — it is refused and nothing stored.
func TestSetPushConfigRejectsSSRFURL(t *testing.T) {
	d := newDispatcher()
	d.store(&Task{ID: "t1", ContextID: "c1", Status: TaskStatus{State: stateCompleted}})

	params, _ := json.Marshal(map[string]any{
		"id":                     "t1",
		"pushNotificationConfig": map[string]any{"url": "http://169.254.169.254/latest/meta-data"},
	})
	rr := httptest.NewRecorder()
	d.setPushConfig(rr, rpcRequest{JSONRPC: "2.0", ID: json.RawMessage("1"), Params: params})

	var resp rpcResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != errInvalidParams {
		t.Fatalf("response = %+v, want invalid-params rejection", resp)
	}
	d.mu.Lock()
	_, stored := d.pushConfigs["t1"]
	d.mu.Unlock()
	if stored {
		t.Error("SSRF callback url must not be stored")
	}
}

// TestDeliverPushBlocksInternalByDefault: even if a config for an internal URL
// slips into the map, deliverPush must not POST to it under the default policy.
func TestDeliverPushBlocksInternalByDefault(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }))
	defer srv.Close() // srv.URL is http://127.0.0.1:PORT — loopback, must be blocked

	d := newDispatcher()
	task := &Task{ID: "t1", Status: TaskStatus{State: stateCompleted}}
	d.pushConfigs["t1"] = PushNotificationConfig{URL: srv.URL}

	d.deliverPush("t1", task)
	if hit {
		t.Error("deliverPush reached a loopback callback under the default policy")
	}
}

// TestAllowPushURLOverrideDelivers: an operator policy can authorize a trusted
// (here loopback) receiver, and delivery then goes through.
func TestAllowPushURLOverrideDelivers(t *testing.T) {
	done := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "application/json" {
			done <- struct{}{}
		}
	}))
	defer srv.Close()

	g := New(Options{AllowPushURL: func(*url.URL) error { return nil }})
	d := g.disp
	if d.guardPushDial {
		t.Fatal("custom AllowPushURL should disable the dial guard")
	}
	task := &Task{ID: "t1", Status: TaskStatus{State: stateCompleted}}
	d.pushConfigs["t1"] = PushNotificationConfig{URL: srv.URL}

	d.deliverPush("t1", task)
	select {
	case <-done:
	default:
		t.Error("operator-authorized callback was not delivered")
	}
}
