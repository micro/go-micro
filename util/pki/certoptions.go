package pki

import (
	"crypto/ed25519"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

// CertOptions are passed to cert options
type CertOptions struct {
	IsCA         bool
	Subject      pkix.Name
	DNSNames     []string
	IPAddresses  []net.IP
	SerialNumber *big.Int
	NotBefore    time.Time
	NotAfter     time.Time

	Parent *x509.Certificate
	Pub    ed25519.PublicKey
	Priv   ed25519.PrivateKey
}

// CertOption sets CertOptions
type CertOption func(c *CertOptions)

// Subject sets the Subject field
func Subject(subject pkix.Name) CertOption {
	return func(c *CertOptions) {
		c.Subject = subject
	}
}

// IsCA states the cert is a CA
func IsCA() CertOption {
	return func(c *CertOptions) {
		c.IsCA = true
	}
}

// DNSNames is a list of hosts to sign in to the certificate
func DNSNames(names ...string) CertOption {
	return func(c *CertOptions) {
		c.DNSNames = names
	}
}

// IPAddresses is a list of IPs to sign in to the certificate
func IPAddresses(ips ...net.IP) CertOption {
	return func(c *CertOptions) {
		c.IPAddresses = ips
	}
}

// KeyPair is the key pair to sign the certificate with
func KeyPair(pub ed25519.PublicKey, priv ed25519.PrivateKey) CertOption {
	return func(c *CertOptions) {
		c.Pub = pub
		c.Priv = priv
	}
}

// SerialNumber is the Certificate Serial number
func SerialNumber(serial *big.Int) CertOption {
	return func(c *CertOptions) {
		c.SerialNumber = serial
	}
}

// NotBefore is the time the certificate is not valid before
func NotBefore(time time.Time) CertOption {
	return func(c *CertOptions) {
		c.NotBefore = time
	}
}

// NotAfter is the time the certificate is not valid after
func NotAfter(time time.Time) CertOption {
	return func(c *CertOptions) {
		c.NotAfter = time
	}
}
