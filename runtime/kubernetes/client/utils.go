package client

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
)

// DefaultService returns default micro kubernetes service definition
func DefaultService(name, version string) *Service {
	Labels := map[string]string{
		"name":    name,
		"micro":   "service",
		"version": version,
	}

	Metadata := &Metadata{
		Name:      name,
		Namespace: "default",
		Version:   version,
		Labels:    Labels,
	}

	Spec := &ServiceSpec{
		Type:     "ClusterIP",
		Selector: Labels,
		Ports: []ServicePort{{
			name + "-port", 9090, "",
		}},
	}

	return &Service{
		Metadata: Metadata,
		Spec:     Spec,
	}
}

// DefaultService returns default micro kubernetes deployment definition
func DefaultDeployment(name, version string) *Deployment {
	Labels := map[string]string{
		"name":    name,
		"micro":   "service",
		"version": version,
	}

	Metadata := &Metadata{
		Name:      name,
		Namespace: "default",
		Version:   version,
		Labels:    Labels,
	}

	Env := []EnvVar{
		{"MICRO_BROKER", "nats"},
		{"MICRO_BROKER_ADDRESS", "nats-cluster"},
		{"MICRO_REGISTRY", "etcd"},
		{"MICRO_REGISTRY_ADDRESS", "etcd-cluster"},
		{"MICRO_PROXY", "go.micro.proxy"},
	}

	// TODO: change the image name here
	Spec := &DeploymentSpec{
		Replicas: 1,
		Selector: &LabelSelector{
			MatchLabels: Labels,
		},
		Template: &Template{
			Metadata: Metadata,
			PodSpec: &PodSpec{
				Containers: []Container{{
					Name:  name,
					Image: DefaultImage,
					Env:   Env,
					Ports: []ContainerPort{{
						Name:          name + "-port",
						ContainerPort: 8080,
					}},
				}},
			},
		},
	}

	return &Deployment{
		Metadata: Metadata,
		Spec:     Spec,
	}
}

// COPIED FROM
// https://github.com/kubernetes/kubernetes/blob/7a725418af4661067b56506faabc2d44c6d7703a/pkg/util/crypto/crypto.go

// CertPoolFromFile returns an x509.CertPool containing the certificates in the given PEM-encoded file.
// Returns an error if the file could not be read, a certificate could not be parsed, or if the file does not contain any certificates
func CertPoolFromFile(filename string) (*x509.CertPool, error) {
	certs, err := certificatesFromFile(filename)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool, nil
}

// certificatesFromFile returns the x509.Certificates contained in the given PEM-encoded file.
// Returns an error if the file could not be read, a certificate could not be parsed, or if the file does not contain any certificates
func certificatesFromFile(file string) ([]*x509.Certificate, error) {
	if len(file) == 0 {
		return nil, errors.New("error reading certificates from an empty filename")
	}
	pemBlock, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	certs, err := CertsFromPEM(pemBlock)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %s", file, err)
	}
	return certs, nil
}

// CertsFromPEM returns the x509.Certificates contained in the given PEM-encoded byte array
// Returns an error if a certificate could not be parsed, or if the data does not contain any certificates
func CertsFromPEM(pemCerts []byte) ([]*x509.Certificate, error) {
	ok := false
	certs := []*x509.Certificate{}
	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		// Only use PEM "CERTIFICATE" blocks without extra headers
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return certs, err
		}

		certs = append(certs, cert)
		ok = true
	}

	if !ok {
		return certs, errors.New("could not read any certificates")
	}
	return certs, nil
}
