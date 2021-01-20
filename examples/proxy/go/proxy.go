package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/asim/go-micro/v3/registry"
)

var (
	registryURI = "http://localhost:8081/registry"
	proxyURI    = "http://localhost:8081"
)

func register(s *registry.Service) {
	b, _ := json.Marshal(s)
	http.Post(registryURI, "application/json", bytes.NewReader(b))
}

func deregister(s *registry.Service) {
	b, _ := json.Marshal(s)
	req, _ := http.NewRequest("DELETE", registryURI, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

func rpcCall(path string, req map[string]interface{}) (string, error) {
	b, _ := json.Marshal(req)
	rsp, err := http.Post(proxyURI+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()
	b, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func httpCall(path string, req url.Values) (string, error) {
	rsp, err := http.PostForm(proxyURI+path, req)
	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
