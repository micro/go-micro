// Package pki provides PKI all the PKI functions necessary to run micro over an untrusted network
// including a CA
package pki

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
)

// GenerateKey returns an ed25519 key
func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// CA generates a self signed CA and returns cert, key in PEM format
func CA(opts ...CertOption) ([]byte, []byte, error) {
	opts = append(opts, IsCA())
	options := CertOptions{}
	for _, o := range opts {
		o(&options)
	}
	template := &x509.Certificate{
		SignatureAlgorithm:    x509.PureEd25519,
		Subject:               options.Subject,
		DNSNames:              options.DNSNames,
		IPAddresses:           options.IPAddresses,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		NotBefore:             options.NotBefore,
		NotAfter:              options.NotAfter,
		SerialNumber:          options.SerialNumber,
		BasicConstraintsValid: true,
	}
	if options.IsCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}
	x509Cert, err := x509.CreateCertificate(rand.Reader, template, template, options.Pub, options.Priv)
	if err != nil {
		return nil, nil, err
	}
	cert, key := &bytes.Buffer{}, &bytes.Buffer{}
	if err := pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: x509Cert}); err != nil {
		return nil, nil, err
	}
	x509Key, err := x509.MarshalPKCS8PrivateKey(options.Priv)
	if err != nil {
		return nil, nil, err
	}
	if err := pem.Encode(key, &pem.Block{Type: "PRIVATE KEY", Bytes: x509Key}); err != nil {
		return nil, nil, err
	}

	return cert.Bytes(), key.Bytes(), nil
}

// CSR generates a certificate request in PEM format
func CSR(opts ...CertOption) ([]byte, error) {
	options := CertOptions{}
	for _, o := range opts {
		o(&options)
	}
	csrTemplate := &x509.CertificateRequest{
		Subject:            options.Subject,
		SignatureAlgorithm: x509.PureEd25519,
		DNSNames:           options.DNSNames,
		IPAddresses:        options.IPAddresses,
	}
	out := &bytes.Buffer{}
	csr, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, options.Priv)
	if err != nil {
		return nil, err
	}
	if err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// Sign decodes a CSR and signs it with the CA
func Sign(CACrt, CAKey, CSR []byte, opts ...CertOption) ([]byte, error) {
	options := CertOptions{}
	for _, o := range opts {
		o(&options)
	}
	asn1CACrt, err := decodePEM(CACrt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode CA Crt PEM")
	}
	if len(asn1CACrt) != 1 {
		return nil, errors.Errorf("expected 1 CA Crt, got %d", len(asn1CACrt))
	}
	caCrt, err := x509.ParseCertificate(asn1CACrt[0].Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "ca is not a valid certificate")
	}
	asn1CAKey, err := decodePEM(CAKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode CA  Key PEM")
	}
	if len(asn1CAKey) != 1 {
		return nil, errors.Errorf("expected 1 CA Key, got %d", len(asn1CACrt))
	}
	caKey, err := x509.ParsePKCS8PrivateKey(asn1CAKey[0].Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "ca key is not a valid private key")
	}
	asn1CSR, err := decodePEM(CSR)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode CSR PEM")
	}
	if len(asn1CSR) != 1 {
		return nil, errors.Errorf("expected 1 CSR, got %d", len(asn1CSR))
	}
	csr, err := x509.ParseCertificateRequest(asn1CSR[0].Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "csr is invalid")
	}
	template := &x509.Certificate{
		SignatureAlgorithm:    x509.PureEd25519,
		Subject:               csr.Subject,
		DNSNames:              csr.DNSNames,
		IPAddresses:           csr.IPAddresses,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		NotBefore:             options.NotBefore,
		NotAfter:              options.NotAfter,
		SerialNumber:          options.SerialNumber,
		BasicConstraintsValid: true,
	}

	x509Cert, err := x509.CreateCertificate(rand.Reader, template, caCrt, caCrt.PublicKey, caKey)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't sign certificate")
	}
	out := &bytes.Buffer{}
	if err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: x509Cert}); err != nil {
		return nil, errors.Wrap(err, "couldn't encode cert")
	}
	return out.Bytes(), nil
}

func decodePEM(PEM []byte) ([]*pem.Block, error) {
	var blocks []*pem.Block
	var asn1 *pem.Block
	var rest []byte
	for {
		asn1, rest = pem.Decode(PEM)
		if asn1 == nil {
			return nil, errors.New("PEM is not valid")
		}
		blocks = append(blocks, asn1)
		if len(rest) == 0 {
			break
		}
	}
	return blocks, nil
}
