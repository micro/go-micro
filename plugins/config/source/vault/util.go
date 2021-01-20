package vault

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/micro/go-micro/v2/config/source"
)

func makeMap(kv map[string]interface{}, secretName string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// if secret version included
	if kv["data"] != nil && kv["metadata"] != nil {
		kv = kv["data"].(map[string]interface{})
	}

	target := data

	// if secretName defined, wrap secrets under a map
	if secretName != "" {
		path := strings.Split(secretName, "/")
		// find (or create) the location we want to put this value at
		for i, dir := range path {
			if _, ok := target[dir]; !ok {
				target[dir] = make(map[string]interface{})
			}
			if i < len(path)-1 {
				target = target[dir].(map[string]interface{})
			} else {
				target[dir] = kv
			}
		}
	}

	return data, nil
}

func getAddress(options source.Options) string {
	// check if there are any addrs
	a, ok := options.Context.Value(addressKey{}).(string)
	if ok {
		// check if http protocol is defined
		if a[0] != 'h' {
			addr, port, err := net.SplitHostPort(a)
			if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
				return fmt.Sprintf("https://%s:%s", a, "8200")
			} else if err == nil {
				return fmt.Sprintf("https://%s:%s", addr, port)
			}
		} else {
			u, _ := url.Parse(a)
			if host, _, _ := net.SplitHostPort(u.Host); host == "" {
				return fmt.Sprintf("%s://%s:%s", u.Scheme, u.Host, "8200")
			} else {
				return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			}
		}
	}
	return ""
}

func getToken(options source.Options) string {
	token, ok := options.Context.Value(tokenKey{}).(string)
	if ok {
		return token
	}
	return ""
}

func getResourcePath(options source.Options) string {
	path, ok := options.Context.Value(resourcePath{}).(string)
	if ok {
		return path
	}
	return ""
}

func getNameSpace(options source.Options) string {
	ns, ok := options.Context.Value(nameSpace{}).(string)
	if ok {
		return ns
	}
	return ""
}

func getSecretName(options source.Options) string {
	ns, ok := options.Context.Value(secretName{}).(string)
	if ok {
		return ns
	}
	return ""
}
