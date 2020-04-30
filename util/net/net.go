package net

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// HostPort format addr and port suitable for dial
func HostPort(addr string, port interface{}) string {
	host := addr
	if strings.Count(addr, ":") > 0 {
		host = fmt.Sprintf("[%s]", addr)
	}
	// when port is blank or 0, host is a queue name
	if v, ok := port.(string); ok && v == "" {
		return host
	} else if v, ok := port.(int); ok && v == 0 && net.ParseIP(host) == nil {
		return host
	}

	return fmt.Sprintf("%s:%v", host, port)
}

// Listen takes addr:portmin-portmax and binds to the first available port
// Example: Listen("localhost:5000-6000", fn)
func Listen(addr string, fn func(string) (net.Listener, error)) (net.Listener, error) {

	if strings.Count(addr, ":") == 1 && strings.Count(addr, "-") == 0 {
		return fn(addr)
	}

	// host:port || host:min-max
	host, ports, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// try to extract port range
	prange := strings.Split(ports, "-")

	// single port
	if len(prange) < 2 {
		return fn(addr)
	}

	// we have a port range

	// extract min port
	min, err := strconv.Atoi(prange[0])
	if err != nil {
		return nil, errors.New("unable to extract port range")
	}

	// extract max port
	max, err := strconv.Atoi(prange[1])
	if err != nil {
		return nil, errors.New("unable to extract port range")
	}

	// range the ports
	for port := min; port <= max; port++ {
		// try bind to host:port
		ln, err := fn(HostPort(host, port))
		if err == nil {
			return ln, nil
		}

		// hit max port
		if port == max {
			return nil, err
		}
	}

	// why are we here?
	return nil, fmt.Errorf("unable to bind to %s", addr)
}

// Proxy returns the proxy and the address if it exits
func Proxy(service string, address []string) (string, []string, bool) {
	var hasProxy bool

	// get proxy
	if prx := os.Getenv("MICRO_PROXY"); len(prx) > 0 {
		// default name
		if prx == "service" {
			prx = "go.micro.proxy"
		}
		service = prx
		hasProxy = true
	}

	// get proxy address
	if prx := os.Getenv("MICRO_PROXY_ADDRESS"); len(prx) > 0 {
		address = []string{prx}
		hasProxy = true
	}

	if prx := os.Getenv("MICRO_NETWORK"); len(prx) > 0 {
		// default name
		if prx == "service" {
			prx = "go.micro.network"
		}
		service = prx
		hasProxy = true
	}

	if prx := os.Getenv("MICRO_NETWORK_ADDRESS"); len(prx) > 0 {
		address = []string{prx}
		hasProxy = true
	}

	return service, address, hasProxy
}
