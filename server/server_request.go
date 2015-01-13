package server

import (
	"net/http"
)

type serverRequest struct {
	req *http.Request
}

func (s *serverRequest) Headers() Headers {
	return s.req.Header
}

func (s *serverRequest) Session(name string) string {
	if sess := s.Headers().Get(name); len(sess) > 0 {
		return sess
	}

	c, err := s.req.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
