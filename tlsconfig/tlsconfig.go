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
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/lstoll/grpce/identitydoc"
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
// AES128 & SHA256 preferred over AES256 & SHA384:
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
func NewMutualTLS(caCerts [][]byte, serverCert tls.Certificate) (*tls.Config, error) {
	pool := x509.NewCertPool()
	for _, caCert := range caCerts {
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to append CA cert")
		}
	}
	cfg := New()
	cfg.ClientAuth = tls.RequireAndVerifyClientCert
	cfg.Certificates = []tls.Certificate{serverCert}
	cfg.ClientCAs = pool
	cfg.RootCAs = pool
	return cfg, nil
}

const (
	InstanceIdentityDocID int = iota
	InstanceIdentitySigID
)

var (
	oidPrefix = []int{0x0, 0x0, 'D', 0x0, 'G', 'E'}

	InstanceIdentityDocOID asn1.ObjectIdentifier = append(oidPrefix, InstanceIdentityDocID)
	InstanceIdentitySigOID asn1.ObjectIdentifier = append(oidPrefix, InstanceIdentitySigID)
)

var ErrCannotAppendFromPEM = errors.New("cannot append from PEM")

// CA is a certificate & key that generate new signed leaf TLS Certificates.
type CA tls.Certificate

// LoadCA initializes a TLS certificate and key, along with an optional
// certificate chain from raw PEM encoded values.
func LoadCA(certPEM, keyPEM []byte, chainPEMs ...[]byte) (*CA, error) {
	kp, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	if kp.Leaf, err = x509.ParseCertificate(kp.Certificate[0]); err != nil {
		return nil, err
	}

	ca := CA(kp)
	for _, chainPEM := range chainPEMs {
		var chainDER *pem.Block
		for {
			chainDER, chainPEM = pem.Decode(chainPEM)
			if chainDER == nil {
				break
			}

			ca.Certificate = append(ca.Certificate, chainDER.Bytes)
		}
	}

	return &ca, nil
}

type LeafConfig struct {
	// Hostname is used for the subject CN and DNSNames fields. Ignored if CSR is present.
	Hostname string
	// CSR is the x509 certificate request.
	CSR *x509.CertificateRequest
	// IID is the EC2 Instance Identity Document data and signature.
	IID *identitydoc.InstanceIdentityDocument
	// PublicKeyAlgorithm is the type of public key generated for the certificate.
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

// NewLeaf generates a new leaf certificate & key signed by c.
func (c *CA) NewLeaf(config LeafConfig) (*tls.Certificate, error) {
	sn, err := serialNumber()
	if err != nil {
		return nil, err
	}

	unsignedCert := &x509.Certificate{
		BasicConstraintsValid: true,
		SerialNumber:          sn,
		SubjectKeyId:          sn.Bytes(),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
			Organization: []string{"Heroku"},
			CommonName:   config.Hostname,
		},
		NotBefore: time.Now().Add(-5 * time.Minute),
		NotAfter:  time.Now().Add(3 * 8760 * time.Hour), // 3 years
		DNSNames:  []string{config.Hostname},
	}

	var (
		privateKey crypto.PrivateKey
		publicKey  crypto.PublicKey
	)

	switch config.PublicKeyAlgorithm {
	case x509.RSA:
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}

		privateKey, publicKey = priv, &priv.PublicKey
	case x509.ECDSA, x509.UnknownPublicKeyAlgorithm:
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}

		privateKey, publicKey = priv, &priv.PublicKey
	default:
		return nil, errors.Errorf("unsupported x509 public key algorithm: %s", string(config.PublicKeyAlgorithm))
	}

	if csr := config.CSR; csr != nil {
		privateKey, publicKey = nil, csr.PublicKey

		unsignedCert.Subject.CommonName = csr.Subject.CommonName
		unsignedCert.DNSNames = csr.DNSNames
		unsignedCert.IPAddresses = csr.IPAddresses
	}

	if iid := config.IID; iid != nil {
		if err := iid.CheckSignature(); err != nil {
			return nil, err
		}

		unsignedCert.ExtraExtensions = []pkix.Extension{
			pkix.Extension{
				Id:    InstanceIdentityDocOID,
				Value: b64(gz(iid.Doc)),
			},
			pkix.Extension{
				Id:    InstanceIdentitySigOID,
				Value: b64(iid.Sig),
			},
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, unsignedCert, c.Leaf, publicKey, c.PrivateKey)
	if err != nil {
		return nil, err
	}

	signedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: append([][]byte{certDER}, c.Certificate...),
		PrivateKey:  privateKey,
		Leaf:        signedCert,
	}, nil
}

// PoolFromPEM accepts a RootCA PEM in the form of a byte slice and returns a cert pool.
func PoolFromPEM(cert []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(cert) {
		return nil, ErrCannotAppendFromPEM
	}
	return pool, nil
}

func serialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}

func gz(p []byte) []byte {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		panic("impossible")
	}

	if _, err := w.Write(p); err != nil {
		panic("impossible")
	}
	if err := w.Close(); err != nil {
		panic("impossible")
	}
	return b.Bytes()
}

func b64(p []byte) []byte {
	buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(p)))
	base64.RawStdEncoding.Encode(buf, p)
	return buf
}
