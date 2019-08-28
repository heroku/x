package mustcert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
)

// Pool is a set of x509 certificates.
func Pool(certs ...*tls.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, cert := range certs {
		for _, certData := range cert.Certificate {
			block := &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certData,
			}

			if !pool.AppendCertsFromPEM(pem.EncodeToMemory(block)) {
				panic("AppendCertsFromPEM failed")
			}
		}
	}
	return pool
}
