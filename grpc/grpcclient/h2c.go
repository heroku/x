package grpcclient

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/lstoll/grpce/h2c"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// DialH2C will establish a connection to the grpcserver.NewStandardH2C server
// listening at the passed in URL, and return a connection for it. This will
// _not_ register it in the client registry, that is left to the user.
func DialH2C(serverURL string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ou, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing provided URL")
	}

	port := ou.Port()
	if ou.Port() == "" {
		p, err := net.LookupPort("tcp", ou.Scheme)
		if err != nil {
			return nil, fmt.Errorf("unable to determine default port for scheme %s", ou.Scheme)
		}
		port = strconv.Itoa(p)
	}

	opts = append(opts, []grpc.DialOption{
		//TODO: SA1019: grpc.WithDialer is deprecated: use WithContextDialer instead  (staticcheck)
		grpc.WithDialer(h2c.Dialer{URL: ou}.DialGRPC), //nolint:staticcheck
		// TLS is done at the HTTP/1.1 level, so we never know....
		grpc.WithInsecure(),
	}...)

	conn, err := grpc.Dial(
		net.JoinHostPort(ou.Hostname(), port),
		opts...,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error dialing server")
	}
	return conn, nil
}
