package addr

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestIsLocal(t *testing.T) {
	testData := []struct {
		addr   string
		expect bool
	}{
		{"localhost", true},
		{"localhost:8080", true},
		{"127.0.0.1", true},
		{"127.0.0.1:1001", true},
		{"80.1.1.1", false},
	}

	for _, d := range testData {
		res := IsLocal(d.addr)
		if res != d.expect {
			t.Fatalf("expected %t got %t", d.expect, res)
		}
	}
}

func TestExtractor(t *testing.T) {
	testData := []struct {
		addr   string
		expect string
		parse  bool
	}{
		{"127.0.0.1", "127.0.0.1", false},
		{"10.0.0.1", "10.0.0.1", false},
		{"", "", true},
		{"0.0.0.0", "", true},
		{"[::]", "", true},
	}

	for _, d := range testData {
		addr, err := Extract(d.addr)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}

		if d.parse {
			ip := net.ParseIP(addr)
			if ip == nil {
				t.Error("Unexpected nil IP")
			}
		} else if addr != d.expect {
			t.Errorf("Expected %s got %s", d.expect, addr)
		}
	}
}

func TestFindIP(t *testing.T) {
	localhost, _ := net.ResolveIPAddr("ip", "127.0.0.1")
	localhostIPv6, _ := net.ResolveIPAddr("ip", "::1")
	privateIP, _ := net.ResolveIPAddr("ip", "10.0.0.1")
	publicIP, _ := net.ResolveIPAddr("ip", "100.0.0.1")
	publicIPv6, _ := net.ResolveIPAddr("ip", "2001:0db8:85a3:0000:0000:8a2e:0370:7334")

	testCases := []struct {
		addrs  []net.Addr
		ip     net.IP
		errMsg string
	}{
		{
			addrs:  []net.Addr{},
			ip:     nil,
			errMsg: ErrIPNotFound.Error(),
		},
		{
			addrs: []net.Addr{localhost},
			ip:    localhost.IP,
		},
		{
			addrs: []net.Addr{localhost, localhostIPv6},
			ip:    localhost.IP,
		},
		{
			addrs: []net.Addr{localhostIPv6},
			ip:    localhostIPv6.IP,
		},
		{
			addrs: []net.Addr{privateIP, localhost},
			ip:    privateIP.IP,
		},
		{
			addrs: []net.Addr{privateIP, publicIP, localhost},
			ip:    privateIP.IP,
		},
		{
			addrs: []net.Addr{publicIP, privateIP, localhost},
			ip:    privateIP.IP,
		},
		{
			addrs: []net.Addr{publicIP, localhost},
			ip:    publicIP.IP,
		},
		{
			addrs: []net.Addr{publicIP, localhostIPv6},
			ip:    publicIP.IP,
		},
		{
			addrs: []net.Addr{localhostIPv6, publicIP},
			ip:    publicIP.IP,
		},
		{
			addrs: []net.Addr{localhostIPv6, publicIPv6, publicIP},
			ip:    publicIPv6.IP,
		},
		{
			addrs: []net.Addr{publicIP, publicIPv6},
			ip:    publicIP.IP,
		},
	}

	for _, tc := range testCases {
		ip, err := findIP(tc.addrs)
		if tc.errMsg == "" {
			assert.Nil(t, err)
			assert.Equal(t, tc.ip.String(), ip.String())
		} else {
			assert.NotNil(t, err)
			assert.Equal(t, tc.errMsg, err.Error())
		}
	}
}
