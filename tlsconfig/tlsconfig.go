// Package tlsconfig provides a safe set of TLS configurations for the Mozilla
// recommended ciphersuites.
//
// See https://wiki.mozilla.org/Security/Server_Side_TLS
//
// Prioritized by:
//   Key Ex:   ECDHE > DH > RSA
//   Enc:      CHACHA20 > AES-GCM > AES-CBC > 3DES
//   MAC:      AEAD > SHA256 > SHA384 > SHA1 (SHA)
//   AES:      128 > 256
//   Cert Sig: ECDSA > RSA
//
// Modern:  strongest ciphers (PFS-only) & latest TLS version(s)
// Default: mix of various strength ciphers & recent TLS versions
// Strict:  deprecated, Default plus ECDHE+RSA+AES{128,256}+CBC+SHA1 for IE 11
// Legacy:  many ciphers & TLS versions for maximum compatibility, less secure
package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
)

var (
	// DefaultCiphers provides strong security for a wide range of clients.
	DefaultCiphers = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,  // 0xcca9
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,    // 0xcca8
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, // 0xc02b
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,   // 0xc02f
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, // 0xc02c
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,   // 0xc030
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, // 0xc023
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,   // 0xc027
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,         // 0x009c
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,         // 0x003c
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,         // 0x009d
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,      // 0xc013
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,      // 0xc014
	}

	// LegacyCiphers supports a maximum number of old devices.
	//
	// See https://wiki.mozilla.org/Security/Server_Side_TLS#Old_backward_compatibility
	LegacyCiphers = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, // 0xc02f
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, // 0xc027
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,    // 0xc013
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, // 0xc030
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,    // 0xc014
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,       // 0x009c
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,       // 0x003c
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,          // 0x002f
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,       // 0x009d
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,          // 0x0035
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,         // 0x000a
	}

	// ModernCiphers provides the highest level of security for modern devices.
	ModernCiphers = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,  // 0xcca9
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,    // 0xcca8
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, // 0xc02b
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,   // 0xc02f
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, // 0xc02c
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,   // 0xc030
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, // 0xc023
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,   // 0xc027
	}

	// StrictCiphers balences high level of security with backwards compatibility.
	StrictCiphers = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,  // 0xcca9
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,    // 0xcca8
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, // 0xc02b
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,   // 0xc02f
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, // 0xc02c
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,   // 0xc030
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, // 0xc023
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,   // 0xc027
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,      // 0xc013
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,      // 0xc014
	}
)

// Legacy modifies config with safe defaults for backwards compatibility.
func Legacy(config *tls.Config) {
	config.CipherSuites = LegacyCiphers
	config.MinVersion = tls.VersionTLS10
	config.PreferServerCipherSuites = true
}

// Default modifies config with safe defaults for standard compatibility.
func Default(config *tls.Config) {
	config.CipherSuites = DefaultCiphers
	config.MinVersion = tls.VersionTLS11
	config.PreferServerCipherSuites = true
}

// Modern modifies config with safe defaults for modern browser compatibility.
func Modern(config *tls.Config) {
	config.CipherSuites = ModernCiphers
	config.MinVersion = tls.VersionTLS12
	config.PreferServerCipherSuites = true
}

// Strict modifies config with safe defaults for compliance compatibility.
func Strict(config *tls.Config) {
	config.CipherSuites = StrictCiphers
	config.MinVersion = tls.VersionTLS13
	config.PreferServerCipherSuites = true
}

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
		MinVersion:   tls.VersionTLS12,
		CipherSuites: ModernCiphers,
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
