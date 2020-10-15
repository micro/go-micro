package pki

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrivateKey(t *testing.T) {
	_, _, err := GenerateKey()
	assert.NoError(t, err)
}

func TestCA(t *testing.T) {
	pub, priv, err := GenerateKey()
	assert.NoError(t, err)

	serialNumberMax := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberMax)
	assert.NoError(t, err, "Couldn't generate serial")

	cert, key, err := CA(
		KeyPair(pub, priv),
		Subject(pkix.Name{
			Organization: []string{"test"},
		}),
		DNSNames("localhost"),
		IPAddresses(net.ParseIP("127.0.0.1")),
		SerialNumber(serialNumber),
		NotBefore(time.Now().Add(time.Minute*-1)),
		NotAfter(time.Now().Add(time.Minute)),
	)
	assert.NoError(t, err, "Couldn't sign CA")
	asn1Key, _ := pem.Decode(key)
	assert.NotNil(t, asn1Key, "Couldn't decode key")
	assert.Equal(t, "PRIVATE KEY", asn1Key.Type)
	decodedKey, err := x509.ParsePKCS8PrivateKey(asn1Key.Bytes)
	assert.NoError(t, err, "Couldn't decode ASN1 Key")
	assert.Equal(t, priv, decodedKey.(ed25519.PrivateKey))

	pool := x509.NewCertPool()
	assert.True(t, pool.AppendCertsFromPEM(cert), "Coudn't parse cert")

	asn1Cert, _ := pem.Decode(cert)
	assert.NotNil(t, asn1Cert, "Couldn't parse pem cert")
	x509cert, err := x509.ParseCertificate(asn1Cert.Bytes)
	assert.NoError(t, err, "Couldn't parse asn1 cert")
	chains, err := x509cert.Verify(x509.VerifyOptions{
		Roots: pool,
	})
	assert.NoError(t, err, "Cert didn't verify")
	assert.Len(t, chains, 1, "CA should have 1 cert in chain")
}

func TestCSR(t *testing.T) {
	pub, priv, err := GenerateKey()
	assert.NoError(t, err)
	csr, err := CSR(
		Subject(
			pkix.Name{
				CommonName:         "testnode",
				Organization:       []string{"microtest"},
				OrganizationalUnit: []string{"super-testers"},
			},
		),
		DNSNames("localhost"),
		IPAddresses(net.ParseIP("127.0.0.1")),
		KeyPair(pub, priv),
	)
	assert.NoError(t, err, "CSR couldn't be encoded")

	asn1csr, _ := pem.Decode(csr)
	assert.NotNil(t, asn1csr)
	decodedcsr, err := x509.ParseCertificateRequest(asn1csr.Bytes)
	assert.NoError(t, err)
	expected := pkix.Name{
		CommonName:         "testnode",
		Organization:       []string{"microtest"},
		OrganizationalUnit: []string{"super-testers"},
	}
	assert.Equal(t, decodedcsr.Subject.String(), expected.String())
}
