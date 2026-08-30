package mucp

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v6/codec"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/transport"
	"go-micro.dev/v6/transport/headers"
)

func cloneMessage(m *transport.Message) *transport.Message {
	out := &transport.Message{Body: append([]byte(nil), m.Body...)}
	if m.Header != nil {
		out.Header = make(map[string]string, len(m.Header))
		for k, v := range m.Header {
			out.Header[k] = v
		}
	}
	return out
}

// memConn is one end of an in-memory connection pair. A Send on either end
// is delivered to the other end's Recv.
type memConn struct {
	peer *memConn
	recv chan *transport.Message
	done chan struct{}
}

func newMemConnPair() (*memConn, *memConn) {
	a := &memConn{recv: make(chan *transport.Message, 32), done: make(chan struct{})}
	b := &memConn{recv: make(chan *transport.Message, 32), done: make(chan struct{})}
	a.peer = b
	b.peer = a
	return a, b
}

func (c *memConn) Send(m *transport.Message) error {
	select {
	case c.peer.recv <- cloneMessage(m):
		return nil
	case <-c.done:
		return errors.New("conn closed")
	}
}

func (c *memConn) Recv(m *transport.Message) error {
	select {
	case msg := <-c.recv:
		*m = *msg
		return nil
	case <-c.done:
		return errors.New("conn closed")
	}
}

func (c *memConn) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	return nil
}

// recordingConn captures every Send for assertion.
type recordingConn struct {
	mu   sync.Mutex
	tags []*transport.Message
}

func (r *recordingConn) Send(m *transport.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags = append(r.tags, cloneMessage(m))
	return nil
}

func (r *recordingConn) Recv(*transport.Message) error { return nil }

func (r *recordingConn) Close() error { return nil }

func (r *recordingConn) last() *transport.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.tags) == 0 {
		return nil
	}
	return r.tags[len(r.tags)-1]
}

func jsonReq() *transport.Message {
	return &transport.Message{Header: map[string]string{"Content-Type": "application/json"}}
}

func fullReq() *transport.Message {
	return &transport.Message{Header: map[string]string{
		"Content-Type":   "application/json",
		headers.ID:       "1",
		headers.Request:  "Svc",
		headers.Method:   "Method",
		headers.Endpoint: "Endpoint",
	}}
}

type payload struct {
	Name string `json:"name"`
}

// TestRoundTrip drives a full client->server->client cycle over an in-memory
// connection, exercising the server's preloaded first message, the socket
// read path on subsequent messages, and the response read on the client.
func TestRoundTrip(t *testing.T) {
	ca, cb := newMemConnPair()

	cc, err := NewClient(ca, Options{Request: jsonReq(), Protocol: "mucp"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	in := payload{Name: "alice"}
	if err := cc.Write(&codec.Message{Id: "1", Target: "Svc", Method: "Method", Endpoint: "Endpoint"}, in); err != nil {
		t.Fatalf("client Write: %v", err)
	}

	// The server accept loop passes the wire message to the pseudo socket;
	// the codec's first ReadHeader pulls it off the socket. Options.Request
	// only carries the construction-time codec selection.
	sc, err := NewServer(cb, Options{Request: fullReq()})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if sc.String() != "mucp" {
		t.Fatalf("server String() = %q, want mucp", sc.String())
	}

	var hdr codec.Message
	if err := sc.ReadHeader(&hdr, codec.Request); err != nil {
		t.Fatalf("server ReadHeader: %v", err)
	}
	if hdr.Id != "1" || hdr.Target != "Svc" || hdr.Method != "Method" || hdr.Endpoint != "Endpoint" {
		t.Fatalf("server header = %+v, want id=1 target=Svc method=Method endpoint=Endpoint", hdr)
	}
	var got payload
	if err := sc.ReadBody(&got); err != nil {
		t.Fatalf("server ReadBody: %v", err)
	}
	if got != in {
		t.Fatalf("server decoded %+v, want %+v", got, in)
	}

	// Second request exercises the repeated read path on the same codec.
	second := payload{Name: "carol"}
	if err := cc.Write(&codec.Message{Id: "2", Target: "Svc", Method: "Method", Endpoint: "Endpoint"}, second); err != nil {
		t.Fatalf("client second Write: %v", err)
	}
	var hdr2 codec.Message
	if err := sc.ReadHeader(&hdr2, codec.Request); err != nil {
		t.Fatalf("server second ReadHeader: %v", err)
	}
	if hdr2.Id != "2" {
		t.Fatalf("server second header id = %q, want 2", hdr2.Id)
	}
	var got2 payload
	if err := sc.ReadBody(&got2); err != nil {
		t.Fatalf("server second ReadBody: %v", err)
	}
	if got2 != second {
		t.Fatalf("server decoded %+v, want %+v", got2, second)
	}

	// Response back to the client over the socket.
	reply := payload{Name: "bob"}
	if err := sc.Write(&codec.Message{Id: "1"}, reply); err != nil {
		t.Fatalf("server Write: %v", err)
	}
	var rh codec.Message
	if err := cc.ReadHeader(&rh, codec.Response); err != nil {
		t.Fatalf("client ReadHeader (response): %v", err)
	}
	var gotReply payload
	if err := cc.ReadBody(&gotReply); err != nil {
		t.Fatalf("client ReadBody (response): %v", err)
	}
	if gotReply != reply {
		t.Fatalf("client decoded %+v, want %+v", gotReply, reply)
	}
}

// TestClientWritesPlainHeaders verifies the client writes plain Micro-*
// headers and copies the request content type.
func TestClientWritesPlainHeaders(t *testing.T) {
	rc := &recordingConn{}
	cc, err := NewClient(rc, Options{Request: jsonReq(), Protocol: "mucp", Stream: "s1"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := cc.Write(&codec.Message{Id: "1", Target: "Svc", Method: "Method", Endpoint: "Endpoint", Error: "oops"}, payload{Name: "x"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := rc.last()
	if got == nil {
		t.Fatal("nothing sent")
	}
	for h, w := range map[string]string{
		headers.ID:          "1",
		headers.Request:     "Svc",
		headers.Method:      "Method",
		headers.Endpoint:    "Endpoint",
		headers.Error:       "oops",
		headers.Stream:      "s1",
		headers.ContentType: "application/json",
	} {
		if got.Header[h] != w {
			t.Errorf("client header %q = %q, want %q", h, got.Header[h], w)
		}
	}
	// Client must not write X- prefixed headers.
	if _, ok := got.Header["X-"+headers.ID]; ok {
		t.Error("client unexpectedly wrote X-Micro-ID")
	}
}

// TestServerDualWritesHeaders verifies the server writes both plain and X-
// prefixed headers for legacy peers.
func TestServerDualWritesHeaders(t *testing.T) {
	rc := &recordingConn{}
	sc, err := NewServer(rc, Options{Request: fullReq()})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if err := sc.Write(&codec.Message{Id: "1", Target: "Svc", Method: "Method", Endpoint: "Endpoint"}, payload{Name: "x"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := rc.last()
	if got == nil {
		t.Fatal("nothing sent")
	}
	for _, h := range []string{headers.ID, headers.Request, headers.Method, headers.Endpoint} {
		if got.Header[h] == "" {
			t.Errorf("server plain header %q missing", h)
		}
		if got.Header["X-"+h] != got.Header[h] {
			t.Errorf("server X- header %q = %q, want %q", "X-"+h, got.Header["X-"+h], got.Header[h])
		}
	}
	if len(got.Body) == 0 {
		t.Error("server sent empty body")
	}
}

// TestClientReadsXFallback verifies the client tolerates X-Micro-* headers
// from legacy peers.
func TestClientReadsXFallback(t *testing.T) {
	ca, cb := newMemConnPair()
	cc, err := NewClient(ca, Options{Request: jsonReq(), Protocol: "mucp"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := cb.Send(&transport.Message{Header: map[string]string{
		headers.ContentType:     "application/json",
		"X-" + headers.ID:       "9",
		"X-" + headers.Method:   "M",
		"X-" + headers.Endpoint: "E",
	}}); err != nil {
		t.Fatalf("feed response: %v", err)
	}

	var h codec.Message
	if err := cc.ReadHeader(&h, codec.Response); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if h.Id != "9" || h.Method != "M" || h.Endpoint != "E" {
		t.Fatalf("client header = %+v, want id=9 method=M endpoint=E", h)
	}
}

// TestServerReadsPlainPriority: when a message carries both plain and X-
// headers the plain value wins (the X- fallback only fills gaps).
func TestServerReadsPlainPriority(t *testing.T) {
	sock, feeder := newMemConnPair()
	sc, err := NewServer(sock, Options{Request: fullReq()})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := feeder.Send(&transport.Message{Header: map[string]string{
		headers.ContentType:     "application/json",
		headers.ID:              "1",
		headers.Request:         "Svc",
		"X-" + headers.Method:   "xed",
		headers.Method:          "plain",
		"X-" + headers.Endpoint: "xend",
	}}); err != nil {
		t.Fatalf("feed: %v", err)
	}

	var h codec.Message
	if err := sc.ReadHeader(&h, codec.Request); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}
	if h.Method != "plain" {
		t.Fatalf("method = %q, want plain (X-Micro-Method must lose)", h.Method)
	}
	if h.Endpoint != "xend" {
		t.Fatalf("endpoint = %q, want xend (X- fallback fills gap)", h.Endpoint)
	}
}

// TestClientErrorEnvelope: codec errors on the client carry the Domain
// envelope (go.micro.client.codec), matching the historical error domain.
func TestClientErrorEnvelope(t *testing.T) {
	ca, cb := newMemConnPair()
	cc, err := NewClient(ca, Options{Request: jsonReq(), Protocol: "mucp", Domain: "go.micro.client"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := cb.Send(&transport.Message{
		Header: map[string]string{headers.ContentType: "application/json"},
		Body:   []byte("{not-json}\n"),
	}); err != nil {
		t.Fatalf("feed response: %v", err)
	}

	var frame codec.Message
	if err := cc.ReadHeader(&frame, codec.Response); err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	var out payload
	err = cc.ReadBody(&out)
	if err == nil {
		t.Fatalf("ReadBody: want decode error, got nil")
	}
	if s := err.Error(); !strings.Contains(s, "go.micro.client.codec") {
		t.Fatalf("ReadBody error = %q, want go.micro.client.codec envelope", s)
	}
}

// TestLegacyClientRewritesContentType: a client talking to a node with no
// protocol metadata and a json/protobuf content type falls back to the legacy
// json-rpc/proto-rpc wire and rewrites the content type in place.
func TestLegacyClientRewritesContentType(t *testing.T) {
	req := &transport.Message{Header: map[string]string{"Content-Type": "application/json"}}
	if _, err := NewClient(&recordingConn{}, Options{Request: req}); err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if ct := req.Header["Content-Type"]; ct != "application/json-rpc" {
		t.Fatalf("legacy rewrite: content type = %q, want application/json-rpc", ct)
	}

	protoReq := &transport.Message{Header: map[string]string{"Content-Type": "application/protobuf"}}
	if _, err := NewClient(&recordingConn{}, Options{Request: protoReq}); err != nil {
		t.Fatalf("NewClient (protobuf): %v", err)
	}
	if ct := protoReq.Header["Content-Type"]; ct != "application/proto-rpc" {
		t.Fatalf("legacy rewrite: content type = %q, want application/proto-rpc", ct)
	}
}

// TestLegacyClientSkipsRewrite: a node advertising a protocol metadata entry
// keeps the modern content type untouched.
func TestLegacyClientSkipsRewrite(t *testing.T) {
	req := &transport.Message{Header: map[string]string{"Content-Type": "application/json"}}
	if _, err := NewClient(&recordingConn{}, Options{Request: req, Protocol: "mucp"}); err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if ct := req.Header["Content-Type"]; ct != "application/json" {
		t.Fatalf("content type = %q, want application/json", ct)
	}
}

// TestLegacyServerBackfillsMethod: a server frame carrying only an endpoint
// header gets the method backfilled so the router sees a consistent message.
func TestLegacyServerBackfillsMethod(t *testing.T) {
	req := &transport.Message{Header: map[string]string{
		headers.ContentType: "application/json",
		headers.Endpoint:    "Ep",
	}}
	if _, err := NewServer(&recordingConn{}, Options{Request: req}); err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if req.Header[headers.Method] != "Ep" {
		t.Fatalf("backfilled method = %q, want Ep", req.Header[headers.Method])
	}
}

// preloadCodec simulates the grpc codec: String()=="grpc" triggers the
// first-message preload hack, and ReadHeader consumes bytes from the buffer.
type preloadCodec struct {
	rwc      io.ReadWriteCloser
	consumed int
}

func (c *preloadCodec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	buf := make([]byte, 5)
	if _, err := c.rwc.Read(buf); err != nil {
		return err
	}
	c.consumed += len(buf)
	return nil
}

func (c *preloadCodec) ReadBody(b interface{}) error            { return nil }
func (c *preloadCodec) Write(*codec.Message, interface{}) error { return nil }
func (c *preloadCodec) Close() error                            { return nil }
func (c *preloadCodec) String() string                          { return "grpc" }

func preloadFactory(rwc io.ReadWriteCloser) codec.Codec { return &preloadCodec{rwc: rwc} }

// TestGRPCPreload verifies the server side first-message preload: for a grpc
// content type the preloaded request body is served from the buffer without a
// blocking socket read.
func TestGRPCPreload(t *testing.T) {
	req := &transport.Message{
		Header: map[string]string{"Content-Type": "application/grpc"},
		Body:   []byte("0123456789"),
	}
	sc, err := NewServer(&recordingConn{}, Options{Request: req, Codecs: map[string]codec.NewCodec{
		"application/grpc": preloadFactory,
	}})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if sc.String() != "grpc" {
		t.Fatalf("server String() = %q, want grpc", sc.String())
	}

	done := make(chan error, 1)
	go func() {
		var h codec.Message
		done <- sc.ReadHeader(&h, codec.Request)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ReadHeader (preloaded): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadHeader blocked on the socket instead of serving the preloaded body")
	}

	c := sc.(*serverCodec)
	pc := c.codec.(*preloadCodec)
	if pc.consumed != 5 {
		t.Fatalf("grpc codec consumed %d bytes, want 5 (preload from req.Body)", pc.consumed)
	}
}

// failingCodec fails to encode non-nil bodies, mirroring a protobuf oneof
// missing branch. On a nil body it records the message error.
type failingCodec struct {
	rwc      io.ReadWriteCloser
	wroteNil bool
}

const failText = "simulating a codec write failure"

func (c *failingCodec) ReadHeader(*codec.Message, codec.MessageType) error { return nil }
func (c *failingCodec) ReadBody(interface{}) error                         { return nil }
func (c *failingCodec) Close() error                                       { return nil }
func (c *failingCodec) String() string                                     { return "rpc" }

func (c *failingCodec) Write(m *codec.Message, dest interface{}) error {
	if dest == nil {
		c.wroteNil = true
		if m.Error != "" {
			c.rwc.Write([]byte(m.Error))
		}
		return nil
	}
	return errors.New(failText)
}

func failingFactory(rwc io.ReadWriteCloser) codec.Codec { return &failingCodec{rwc: rwc} }

// TestCodecWriteError: when a codec cannot encode a body the server must send
// an error frame instead of leaving the socket open with nothing on the wire.
func TestCodecWriteError(t *testing.T) {
	rc := &recordingConn{}
	req := &transport.Message{Header: map[string]string{
		headers.ContentType: "application/json",
		headers.Request:     "Svc",
		headers.Method:      "Method",
		headers.Endpoint:    "Method",
	}}
	sc, err := NewServer(rc, Options{Request: req, Codecs: map[string]codec.NewCodec{
		"application/json": failingFactory,
	}})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if err := sc.Write(&codec.Message{Id: "0", Endpoint: "Method"}, "body"); err != nil {
		t.Fatalf("Write unexpectedly failed: %v", err)
	}

	got := rc.last()
	if got == nil {
		t.Fatal("nothing sent")
	}
	const wantErr = "Unable to encode body: " + failText
	if got.Header[headers.Error] != wantErr {
		t.Fatalf("error header = %q, want %q", got.Header[headers.Error], wantErr)
	}
	if len(got.Body) != 0 {
		t.Fatalf("error frame body = %q, want empty", got.Body)
	}

	c := sc.(*serverCodec)
	if fc, ok := c.codec.(*failingCodec); !ok || !fc.wroteNil {
		t.Error("codec did not receive the nil-body error write")
	}
}

// TestSetupProtocol exercises the merged legacy protocol detector.
func TestSetupProtocol(t *testing.T) {
	// Node with protocol metadata: modern wire, no legacy codec.
	msg := &transport.Message{Header: map[string]string{"Content-Type": "application/json"}}
	modern := &registry.Node{Metadata: map[string]string{"protocol": "mucp"}}
	if got := SetupProtocol(msg, modern); got != nil {
		t.Fatalf("SetupProtocol(modern node) = %v, want nil", got)
	}

	// Node without protocol metadata, json content type: legacy json-rpc.
	legacy := &registry.Node{}
	legacyMsg := &transport.Message{Header: map[string]string{"Content-Type": "application/json"}}
	if got := SetupProtocol(legacyMsg, legacy); got == nil {
		t.Fatal("SetupProtocol(legacy node) = nil, want legacy codec")
	}
	if ct := legacyMsg.Header["Content-Type"]; ct != "application/json-rpc" {
		t.Fatalf("content type = %q, want application/json-rpc", ct)
	}

	// Server side (nil node): no Micro-* headers means legacy codec.
	serverMsg := &transport.Message{Header: map[string]string{"Content-Type": "application/protobuf"}}
	if got := SetupProtocol(serverMsg, nil); got == nil {
		t.Fatalf("SetupProtocol(nil node) = nil, want legacy codec for protobuf")
	}
}

func TestGetHeader(t *testing.T) {
	md := map[string]string{"Micro-Method": "plain"}
	if got := GetHeader(headers.Method, md); got != "plain" {
		t.Fatalf("GetHeader plain = %q, want plain", got)
	}
	xmd := map[string]string{"X-Micro-Method": "x"}
	if got := GetHeader(headers.Method, xmd); got != "x" {
		t.Fatalf("GetHeader X- fallback = %q, want x", got)
	}
}
