package hkafka

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"strings"

	"github.com/joeshaw/envdecode"
	"github.com/pkg/errors"
)

type Config struct {
	URL           string `env:"KAFKA_URL,required"`
	TrustedCert   string `env:"KAFKA_TRUSTED_CERT,required"`
	ClientCert    string `env:"KAFKA_CLIENT_CERT,required"`
	ClientCertKey string `env:"KAFKA_CLIENT_CERT_KEY,required"`
}

// NewConfigFromEnv extracts the kafka config from the environment
func NewConfigFromEnv() (Config, error) {
	var c Config
	err := envdecode.Decode(&c)
	return c, err
}

// BrokerAddresses extracted from the host:port pairs in the config's URL
func (c Config) BrokerAddresses() ([]string, error) {
	urls := strings.Split(c.URL, ",")
	addrs := make([]string, len(urls))
	for i, v := range urls {
		u, err := url.Parse(v)
		if err != nil {
			return nil, errors.Wrap(err, "parsing broker url")
		}
		addrs[i] = u.Host
	}
	return addrs, nil
}

// TLSConfig derived from this config
func (c Config) TLSConfig() (*tls.Config, error) {
	// Setup root cert
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(c.TrustedCert))
	if !ok {
		return nil, errors.New("unable to parse trusted certs")
	}

	// Setup client certs
	cert, err := tls.X509KeyPair([]byte(c.ClientCert), []byte(c.ClientCertKey))
	if err != nil {
		return nil, errors.Wrap(err, "setting up client cert")
	}

	tc := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		RootCAs:            roots,
	}
	tc.BuildNameToCertificate()

	return &tc, nil
}

// VerifyServers have certs that are valid with this config
func (c Config) VerifyServers() error {
	tc, err := c.TLSConfig()
	if err != nil {
		return errors.Wrap(err, "creating tls config")
	}

	ba, err := c.BrokerAddresses()
	if err != nil {
		return errors.Wrap(err, "constructing broker list")
	}

	for _, b := range ba {
		if err := verifyServerCert(tc, tc.RootCAs, b); err != nil {
			return errors.Wrap(err, "verifying server cert")
		}
	}
	return nil
}

func verifyServerCert(tc *tls.Config, root *x509.CertPool, url string) error {
	conn, err := tls.Dial("tcp", url, tc)
	if err != nil {
		return errors.Wrap(err, "dialing server")
	}
	defer conn.Close()

	sc := conn.ConnectionState().PeerCertificates[0]
	_, err = sc.Verify(x509.VerifyOptions{Roots: root})
	return errors.Wrap(err, "verifying cert")
}
