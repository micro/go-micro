package server

import (
	"fmt"
	"net"
	"reflect"

	"github.com/micro/go-micro/registry"
)

var (
	privateBlocks []*net.IPNet
)

func init() {
	for _, b := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		if _, block, err := net.ParseCIDR(b); err == nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func extractValue(v reflect.Type, d int) *registry.Value {
	if d == 3 {
		return nil
	}
	if v == nil {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	arg := &registry.Value{
		Name: v.Name(),
		Type: v.Name(),
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			val := extractValue(v.Field(i).Type, d+1)
			if val == nil {
				continue
			}
			val.Name = v.Field(i).Name
			arg.Values = append(arg.Values, val)
		}
	case reflect.Slice:
		p := v.Elem()
		if p.Kind() == reflect.Ptr {
			p = p.Elem()
		}
		arg.Type = "[]" + p.Name()
		val := extractValue(v.Elem(), d+1)
		if val != nil {
			arg.Values = append(arg.Values, val)
		}
	}

	return arg
}

func extractEndpoint(method reflect.Method) *registry.Endpoint {
	if method.PkgPath != "" {
		return nil
	}

	var rspType, reqType reflect.Type
	var stream bool
	mt := method.Type

	switch mt.NumIn() {
	case 3:
		reqType = mt.In(1)
		rspType = mt.In(2)
	case 4:
		reqType = mt.In(2)
		rspType = mt.In(3)
	default:
		return nil
	}

	// are we dealing with a stream?
	switch rspType.Kind() {
	case reflect.Func, reflect.Interface:
		stream = true
	}

	request := extractValue(reqType, 0)
	response := extractValue(rspType, 0)

	return &registry.Endpoint{
		Name:     method.Name,
		Request:  request,
		Response: response,
		Metadata: map[string]string{
			"stream": fmt.Sprintf("%v", stream),
		},
	}
}

func extractSubValue(typ reflect.Type) *registry.Value {
	var reqType reflect.Type
	switch typ.NumIn() {
	case 1:
		reqType = typ.In(0)
	case 2:
		reqType = typ.In(1)
	case 3:
		reqType = typ.In(2)
	default:
		return nil
	}
	return extractValue(reqType, 0)
}

func extractAddress(addr string) (string, error) {
	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]") {
		return addr, nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("Failed to get interface addresses! Err: %v", err)
	}

	var ipAddr []byte

	for _, rawAddr := range addrs {
		var ip net.IP
		switch addr := rawAddr.(type) {
		case *net.IPAddr:
			ip = addr.IP
		case *net.IPNet:
			ip = addr.IP
		default:
			continue
		}

		if ip.To4() == nil {
			continue
		}

		if !isPrivateIP(ip.String()) {
			continue
		}

		ipAddr = ip
		break
	}

	if ipAddr == nil {
		return "", fmt.Errorf("No private IP address found, and explicit IP not provided")
	}

	return net.IP(ipAddr).String(), nil
}

func isPrivateIP(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	for _, priv := range privateBlocks {
		if priv.Contains(ip) {
			return true
		}
	}
	return false
}
