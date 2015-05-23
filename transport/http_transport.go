package transport

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

type headerRoundTripper struct {
	r http.RoundTripper
}

type httpTransport struct {
	client *http.Client
}

type httpTransportClient struct {
	ht   *httpTransport
	addr string
}

type httpTransportSocket struct {
	r *http.Request
	w http.ResponseWriter
}

type httpTransportListener struct {
	listener net.Listener
}

func (t *headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Client-Version", "1.0")
	return t.r.RoundTrip(r)
}

func (h *httpTransportClient) Send(m *Message) (*Message, error) {
	header := make(http.Header)

	for k, v := range m.Header {
		header.Set(k, v)
	}

	reqB := bytes.NewBuffer(m.Body)
	defer reqB.Reset()
	buf := &buffer{
		reqB,
	}

	hreq := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   h.addr,
		},
		Header:        header,
		Body:          buf,
		ContentLength: int64(reqB.Len()),
		Host:          h.addr,
	}

	rsp, err := h.ht.client.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range rsp.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	return mr, nil
}

func (h *httpTransportClient) Close() error {
	return nil
}

func (h *httpTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	b, err := ioutil.ReadAll(h.r.Body)
	if err != nil {
		return err
	}

	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range h.r.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	*m = *mr
	return nil
}

func (h *httpTransportSocket) Send(m *Message) error {
	for k, v := range m.Header {
		h.w.Header().Set(k, v)
	}

	_, err := h.w.Write(m.Body)
	return err
}

func (h *httpTransportSocket) Close() error {
	return nil
}

func (h *httpTransportListener) Addr() string {
	return h.listener.Addr().String()
}

func (h *httpTransportListener) Close() error {
	return h.listener.Close()
}

func (h *httpTransportListener) Accept(fn func(Socket)) error {
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn(&httpTransportSocket{
				r: r,
				w: w,
			})
		}),
	}

	return srv.Serve(h.listener)
}

func (h *httpTransport) Dial(addr string) (Client, error) {
	return &httpTransportClient{
		ht:   h,
		addr: addr,
	}, nil
}

func (h *httpTransport) Listen(addr string) (Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &httpTransportListener{
		listener: l,
	}, nil
}

func newHttpTransport(addrs []string, opt ...Option) *httpTransport {
	client := &http.Client{}
	client.Transport = &headerRoundTripper{http.DefaultTransport}

	return &httpTransport{client: client}
}
