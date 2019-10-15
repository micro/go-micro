package certmagic

import (
	"os"
	"testing"

	"github.com/go-acme/lego/v3/providers/dns/cloudflare"
	"github.com/mholt/certmagic"
	"github.com/micro/go-micro/api/server/acme"
	"github.com/micro/go-micro/sync/lock/memory"
)

func TestCertMagic(t *testing.T) {
	if len(os.Getenv("IN_TRAVIS_CI")) != 0 {
		t.Skip("Travis doesn't let us bind :443")
	}
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

	l, err = New(acme.AcceptToS(true),
		acme.CA(acme.LetsEncryptStagingCA),
		acme.ChallengeProvider(p),
	).NewListener()

	if err != nil {
		t.Error(err.Error())
	}
	l.Close()
}

func TestStorageImplementation(t *testing.T) {
	var s certmagic.Storage
	s = &storage{
		lock: memory.NewLock(),
	}
	if err := s.Lock("test"); err != nil {
		t.Error(err)
	}
	s.Unlock("test")
	New(acme.Cache(s))
}
