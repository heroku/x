package grpcclient

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/lstoll/grpce/h2c"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// DialH2CContext will establish a connection to the grpcserver.NewStandardH2C server
// listening at the passed in URL, and return a connection for it. This will
// _not_ register it in the client registry, that is left to the user. By default the call is non-blocking.
// Use WithBlock for a blocking call. In the blocking case, ctx can be used to cancel or expire the pending connection.
func DialH2CContext(ctx context.Context, serverURL string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing provided URL")
	}
	opts = append(opts,
		grpc.WithContextDialer(h2c.Dialer{URL: u}.DialGRPCContext),
		// TLS is done at the HTTP/1.1 level, so we never know....
		grpc.WithInsecure(),
	)

	port := u.Port()
	if u.Port() == "" {
		p, err := net.LookupPort("tcp", u.Scheme)
		if err != nil {
			return nil, fmt.Errorf("unable to determine default port for scheme %s", u.Scheme)
		}
		port = strconv.Itoa(p)
	}

	conn, err := grpc.DialContext(ctx, net.JoinHostPort(u.Hostname(), port), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error dialing server")
	}
	return conn, nil

}

// DialH2C will establish a connection to the grpcserver.NewStandardH2C server
// listening at the passed in URL, and return a connection for it. This will
// _not_ register it in the client registry, that is left to the user.
//
// Deprecated: Use DialH2CContext instead.
func DialH2C(serverURL string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialH2CContext(context.Background(), serverURL, opts...)
}
