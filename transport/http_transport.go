package transport

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

type headerRoundTripper struct {
	r http.RoundTripper
}

type HttpTransport struct {
	client *http.Client
}

type HttpTransportClient struct {
	ht   *HttpTransport
	addr string
}

type HttpTransportSocket struct {
	r *http.Request
	w http.ResponseWriter
}

type HttpTransportServer struct {
	listener net.Listener
}

func (t *headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Client-Version", "1.0")
	return t.r.RoundTrip(r)
}

func (h *HttpTransportClient) Send(m *Message) (*Message, error) {
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
			//		Path:   path,
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

func (h *HttpTransportClient) Close() error {
	return nil
}

func (h *HttpTransportSocket) Recv() (*Message, error) {
	b, err := ioutil.ReadAll(h.r.Body)
	if err != nil {
		return nil, err
	}

	m := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range h.r.Header {
		if len(v) > 0 {
			m.Header[k] = v[0]
		} else {
			m.Header[k] = ""
		}
	}

	return m, nil
}

func (h *HttpTransportSocket) WriteHeader(k string, v string) {
	h.w.Header().Set(k, v)
}

func (h *HttpTransportSocket) Write(b []byte) error {
	_, err := h.w.Write(b)
	return err
}

func (h *HttpTransportServer) Addr() string {
	return h.listener.Addr().String()
}

func (h *HttpTransportServer) Close() error {
	return h.listener.Close()
}

func (h *HttpTransportServer) Serve(fn func(Socket)) error {
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn(&HttpTransportSocket{
				r: r,
				w: w,
			})
		}),
	}

	return srv.Serve(h.listener)
}

func (h *HttpTransport) NewClient(addr string) (Client, error) {
	return &HttpTransportClient{
		ht:   h,
		addr: addr,
	}, nil
}

func (h *HttpTransport) NewServer(addr string) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &HttpTransportServer{
		listener: l,
	}, nil
}

func NewHttpTransport(addrs []string) *HttpTransport {
	client := &http.Client{}
	client.Transport = &headerRoundTripper{http.DefaultTransport}

	return &HttpTransport{client: client}
}
