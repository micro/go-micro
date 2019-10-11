package certmagic

import (
	"testing"

	"github.com/go-acme/lego/v3/providers/dns/cloudflare"
	"github.com/micro/go-micro/api/server/acme"
)

func TestCertMagic(t *testing.T) {
	l, err := New().NewListener()
	if err != nil {
		t.Error(err.Error())
	}
	l.Close()

	c := cloudflare.NewDefaultConfig()
	c.AuthEmail = ""
	c.AuthKey = ""
	c.AuthToken = "test"
	c.ZoneToken = "test"

	p, err := cloudflare.NewDNSProviderConfig(c)
	if err != nil {
		t.Error(err.Error())
	}

	l, err = New(acme.AcceptTLS(true),
		acme.CA(acme.LetsEncryptStagingCA),
		acme.ChallengeProvider(p),
	).NewListener()

	if err != nil {
		t.Error(err.Error())
	}
	l.Close()
}
