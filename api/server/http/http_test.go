package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestHTTPServer(t *testing.T) {
	testResponse := "hello world"

	s := NewServer("localhost:0")

	s.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testResponse)
	}))

	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	rsp, err := http.Get(fmt.Sprintf("http://%s/", s.Address()))
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != testResponse {
		t.Fatalf("Unexpected response, got %s, expected %s", string(b), testResponse)
	}

	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}
