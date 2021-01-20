package nats

import (
	"os"
	"strings"
	"testing"

	"github.com/go-log/log"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/transport"
	"github.com/nats-io/nats.go"
)

var addrTestCases = []struct {
	name        string
	description string
	addrs       map[string]string // expected address : set address
}{
	{
		"transportOption",
		"set broker addresses through a transport.Option",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"natsOption",
		"set broker addresses through the nats.Option",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"default",
		"check if default Address is set correctly",
		map[string]string{
			"nats://127.0.0.1:4222": ""},
	},
}

// This test will check if options (here nats addresses) set through either
// transport.Option or via nats.Option are successfully set.
func TestInitAddrs(t *testing.T) {

	for _, tc := range addrTestCases {
		t.Run(tc.name, func(t *testing.T) {

			var tr transport.Transport
			var addrs []string

			for _, addr := range tc.addrs {
				addrs = append(addrs, addr)
			}

			switch tc.name {
			case "transportOption":
				// we know that there are just two addrs in the dict
				tr = NewTransport(transport.Addrs(addrs[0], addrs[1]))
			case "natsOption":
				nopts := nats.GetDefaultOptions()
				nopts.Servers = addrs
				tr = NewTransport(Options(nopts))
			case "default":
				tr = NewTransport()
			}

			ntport, ok := tr.(*ntport)
			if !ok {
				t.Fatal("Expected broker to be of types *nbroker")
			}
			// check if the same amount of addrs we set has actually been set
			if len(ntport.addrs) != len(tc.addrs) {
				t.Errorf("Expected Addr count = %d, Actual Addr count = %d",
					len(ntport.addrs), len(tc.addrs))
			}

			for _, addr := range ntport.addrs {
				_, ok := tc.addrs[addr]
				if !ok {
					t.Errorf("Expected '%s' has not been set", addr)
				}
			}
		})
	}
}

var listenAddrTestCases = []struct {
	name     string
	address  string
	mustPass bool
}{
	{"default address", server.DefaultAddress, true},
	{"nats.NewInbox", nats.NewInbox(), true},
	{"correct service name", "micro.test.myservice", true},
	{"several space chars", "micro.test.my new service", false},
	{"one space char", "micro.test.my oldservice", false},
	{"empty", "", false},
}

func TestListenAddr(t *testing.T) {

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		log.Logf("NATS_URL is undefined - skipping tests")
		return
	}

	for _, tc := range listenAddrTestCases {
		t.Run(tc.address, func(t *testing.T) {

			nOpts := nats.GetDefaultOptions()
			nOpts.Servers = []string{natsURL}
			nTport := ntport{
				nopts: nOpts,
			}
			trListener, err := nTport.Listen(tc.address)
			if err != nil {
				if tc.mustPass {
					t.Fatalf("%s (%s) is not allowed", tc.name, tc.address)
				}
				// correctly failed
				return
			}
			if trListener.Addr() != tc.address {
				//special case - since an always string will be returned
				if tc.name == "default address" {
					if strings.Contains(trListener.Addr(), "_INBOX.") {
						return
					}
				}
				t.Errorf("expected address %s but got %s", tc.address, trListener.Addr())
			}
		})
	}
}
