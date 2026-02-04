// Package tls provides TLS utilities for go-micro.
package tls

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"sync"
	"time"
)

var (
	// Track if we've already logged the warning to avoid spam
	warningOnce sync.Once
)

// Config returns a TLS config.
//
// BACKWARD COMPATIBILITY: By default, InsecureSkipVerify is true for compatibility
// with existing deployments. This maintains the existing behavior to avoid breaking
// production systems during upgrades.
//
// SECURITY WARNING: The default behavior skips certificate verification. This is
// insecure and vulnerable to man-in-the-middle attacks.
//
// To enable secure certificate verification (RECOMMENDED for production):
//   - Set environment variable: MICRO_TLS_SECURE=true
//   - Use SecureConfig() function directly
//   - Configure TLSConfig with proper certificates
//   - Use a service mesh (Istio, Linkerd) for mTLS
//
// DEPRECATION NOTICE: The insecure default will be changed in a future major version (v6).
// Please migrate to secure mode by setting MICRO_TLS_SECURE=true in your environment.
func Config() *tls.Config {
	// Check environment for explicit secure mode
	if os.Getenv("MICRO_TLS_SECURE") == "true" {
		return &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
	}

	// Log deprecation warning once (only if not in test environment)
	if os.Getenv("IN_TRAVIS_CI") == "" {
		warningOnce.Do(func() {
			log.Println("[SECURITY WARNING] TLS certificate verification is disabled by default. " +
				"This is insecure and will change in v6. " +
				"Set MICRO_TLS_SECURE=true to enable certificate verification.")
		})
	}

	// DEPRECATED: Default remains insecure for backward compatibility
	// This will change in v6 - please migrate to secure mode
	return &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
}

// SecureConfig returns a TLS config with certificate verification enabled.
// Use this when you have proper CA-signed certificates.
func SecureConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}
}

// InsecureConfig returns a TLS config with certificate verification disabled.
// WARNING: Only use for development/testing.
func InsecureConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
}

// Certificate generates a self-signed certificate for the given hosts.
// Note: These certs are for development only. For production, use proper
// CA-signed certificates or a service mesh.
func Certificate(host ...string) (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Micro"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range host {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	// create public key
	certOut := bytes.NewBuffer(nil)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// create private key
	keyOut := bytes.NewBuffer(nil)
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	return tls.X509KeyPair(certOut.Bytes(), keyOut.Bytes())
}
