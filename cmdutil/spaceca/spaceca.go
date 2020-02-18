// Package spaceca provides helpers for setting up TLS from a CA configuration.
package spaceca

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"

	"github.com/heroku/x/tlsconfig"
)

// CA contains information regarding the root CA and intermediary
// space CA which is used to generate a certificate for a service.
type CA struct {
	RootCert []byte
	Cert     []byte
	Key      []byte
}

// NewServerTLSConfig returns a TLS configuration for servers running in a
// space. The configuration includes a certificate valid for the provided
// domain and signed by the space's CA.
func NewServerTLSConfig(domain string, secData map[string][]byte) (*tls.Config, error) {
	ca := CA{
		RootCert: secData["root.crt"],
		Cert:     secData["space.crt"],
		Key:      secData["space.key"],
	}
	cert, err := NewCACertificate(domain, ca)
	if err != nil {
		return nil, errors.Wrap(err, "error generating TLS cert")
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}
	tlsconfig.Modern(config)
	return config, nil
}

// NewCACertificate returns a certificate signed by the space's CA.
func NewCACertificate(domain string, spaceCA CA) (*tls.Certificate, error) {
	rootCACert := spaceCA.RootCert
	spaceCACert := spaceCA.Cert
	spaceCAKey := spaceCA.Key

	ca, err := tlsconfig.LoadCA(spaceCACert, spaceCAKey, rootCACert)
	if err != nil {
		return nil, errors.Wrap(err, "error loading space CA")
	}

	certConfig := tlsconfig.LeafConfig{
		Hostname:           domain,
		PublicKeyAlgorithm: x509.RSA,
	}

	return ca.NewLeaf(certConfig)
}
