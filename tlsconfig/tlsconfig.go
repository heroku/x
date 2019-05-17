package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
)

// New returns a TLS configuration tuned for performance and security based on
// the recommendations in:
// https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
//
// AES128 & SHA256 prefered over AES256 & SHA384:
// https://github.com/ssllabs/research/wiki/SSL-and-TLS-Deployment-Best-Practices#31-avoid-too-much-security
func New() *tls.Config {
	return &tls.Config{
		PreferServerCipherSuites: true,
		// Only use curves that have assembly implementations.
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
}

// NewMutualTLS returns a TLS configuration setup for mutual TLS
// authentication.
func NewMutualTLS(caCerts [][]byte, certPEM, keyPEM []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "creating X509 key pair")
	}
	pool := x509.NewCertPool()
	for _, caCert := range caCerts {
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to append CA cert")
		}
	}
	cfg := New()
	cfg.ClientAuth = tls.RequireAndVerifyClientCert
	cfg.Certificates = []tls.Certificate{cert}
	cfg.ClientCAs = pool
	cfg.RootCAs = pool
	return cfg, nil
}
