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

// Config returns a TLS config with secure defaults.
// If insecureSkipVerify is true, certificate verification is disabled (NOT recommended for production).
func Config(insecureSkipVerify bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
}

// SecureConfig returns a TLS config with secure defaults and certificate verification enabled.
func SecureConfig() *tls.Config {
	return Config(false)
}

// InsecureConfig returns a TLS config with certificate verification disabled.
// WARNING: This should only be used for development/testing.
func InsecureConfig() *tls.Config {
	return Config(true)
}

// ConfigFromEnv returns a TLS config based on the MICRO_TLS_INSECURE environment variable.
// If MICRO_TLS_INSECURE is set to "true", certificate verification is disabled.
// Otherwise, secure defaults are used.
func ConfigFromEnv() *tls.Config {
	insecure := os.Getenv("MICRO_TLS_INSECURE") == "true"
	return Config(insecure)
}

// Certificate generates a self-signed certificate for the given hosts.
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
			Organization: []string{"Acme Co"},
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
