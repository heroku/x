package h2c

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Dialer connects to a HTTP 1.1 server and performs an h2c upgrade to an HTTP2 connection.
type Dialer struct {
	Dialer    *net.Dialer
	TLSConfig *tls.Config
	URL       *url.URL
}

// DialContext connects to the address on the named network using the provided context.
func (d Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	dialfn := http.DefaultTransport.(*http.Transport).DialContext
	if d.Dialer != nil {
		dialfn = d.Dialer.DialContext
	}

	conn, err := dialfn(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	tlsConfig := d.TLSConfig
	if tlsConfig == nil && d.URL != nil && d.URL.Scheme == "https" {
		tlsConfig = &tls.Config{
			ServerName: d.URL.Hostname(),
		}
	}

	if tlsConfig != nil {
		conn = tls.Client(conn, tlsConfig)
	}

	u := "http://" + addr
	if d.TLSConfig != nil {
		u = "https://" + addr
	}
	if d.URL != nil {
		u = d.URL.String()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")

	if err := req.Write(conn); err != nil {
		return nil, err
	}

	br := bufio.NewReader(conn)

	res, err := http.ReadResponse(br, req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusSwitchingProtocols {
		return nil, errors.New("h2c upgrade failed, recieved non 101 response")
	}
	if strings.ToLower(res.Header.Get("Connection")) != "upgrade" {
		return nil, errors.New("h2c upgrade failed, bad Connection header in response")
	}
	if strings.ToLower(res.Header.Get("Upgrade")) != "h2c" {
		return nil, errors.New("h2c upgrade failed, bad Upgrade header in response")
	}
	if buf, err := ioutil.ReadAll(res.Body); len(buf) > 0 || err != nil {
		return nil, errors.New("h2c upgrade failed, upgrade response body was non empty")
	}

	return &bufferedConn{br: br, Conn: conn}, nil
}

// Dial connects to the address on the named network.
func (d Dialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

// DialGRPC connects to the address before timeout.
// Deprecated: use DialGRPCContext and grpc.WithContextDialer.
func (d Dialer) DialGRPC(addr string, timeout time.Duration) (net.Conn, error) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return d.DialContext(ctx, "tcp", addr)
}

// DialGRPCContext implements the interface required by grpc.WithContextDialer.
func (d Dialer) DialGRPCContext(ctx context.Context, addr string) (net.Conn, error) {
	return d.DialContext(ctx, "tcp", addr)
}

type bufferedConn struct {
	br *bufio.Reader
	net.Conn
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.br.Read(p)
}
