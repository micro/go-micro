package http

import (
	"fmt"
	"github.com/asim/go-micro/v3/api/server"
	"github.com/asim/go-micro/v3/api/server/cors"
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

func TestCORSHTTPServer(t *testing.T) {
	testResponse := "hello world"
	testAllowOrigin := "*"
	testAllowCredentials := true
	testAllowMethods := "GET"
	testAllowHeaders := "Accept, Content-Type, Content-Length"

	s := NewServer("localhost:0",
		server.EnableCORS(true),
		server.CORSConfig(&cors.Config{
			AllowCredentials: testAllowCredentials,
			AllowOrigin:      testAllowOrigin,
			AllowMethods:     testAllowMethods,
			AllowHeaders:     testAllowHeaders,
		}),
	)

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

	allowCredentials := rsp.Header.Get("Access-Control-Allow-Credentials")
	getTestCredentialsStr := func() string {
		if testAllowCredentials == true {
			return "true"
		} else {
			return "false"
		}
	}
	if getTestCredentialsStr() != allowCredentials {
		t.Fatalf("Unexpected Access-Control-Allow-Credentials, got %s, expected %s", allowCredentials, getTestCredentialsStr())
	}

	allowOrigin := rsp.Header.Get("Access-Control-Allow-Origin")
	if testAllowOrigin != allowOrigin {
		t.Fatalf("Unexpected Access-Control-Allow-Origins, got %s, expected %s", allowOrigin, testAllowOrigin)
	}

	allowMethods := rsp.Header.Get("Access-Control-Allow-Methods")
	if testAllowMethods != allowMethods {
		t.Fatalf("Unexpected Access-Control-Allow-Methods, got %s, expected %s", allowMethods, testAllowMethods)
	}
	allowHeaders := rsp.Header.Get("Access-Control-Allow-Headers")
	if testAllowHeaders != allowHeaders {
		t.Fatalf("Unexpected Access-Control-Allow-Headers, got %s, expected %s", allowHeaders, testAllowHeaders)
	}

	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}
