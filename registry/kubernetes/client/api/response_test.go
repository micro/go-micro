package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

func TestResponseReleasesBody(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		wantErr error
	}{
		{name: "ok", status: http.StatusOK},
		{name: "not found", status: http.StatusNotFound, wantErr: ErrNotFound},
		{name: "server error", status: http.StatusInternalServerError, wantErr: ErrOther},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader(`{"items":[]}`)}

			var out map[string]any
			err := newResponse(&http.Response{StatusCode: tc.status, Body: body}, nil).Decode(&out)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if !body.closed {
				t.Fatal("response body was not closed")
			}
		})
	}
}
