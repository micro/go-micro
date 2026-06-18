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
	"math/big"
	"net"
	"os"
	"time"
)

// Config returns the default TLS config.
//
// As of v6, certificate verification is ON by default (secure by default).
// This is the safe choice now that an agent — not just a human on a
// trusted network — can reach an endpoint.
//
// For development against self-signed certificates, set
// MICRO_TLS_INSECURE=true to skip verification, or call InsecureConfig()
// directly. (In v5 the default was the reverse: insecure unless
// MICRO_TLS_SECURE=true was set. That env var is no longer needed.)
func Config() *tls.Config {
	// Opt out of verification for local/dev against self-signed certs.
	if os.Getenv("MICRO_TLS_INSECURE") == "true" {
		return &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	}

	// Secure by default.
	return &tls.Config{
		InsecureSkipVerify: false,
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
