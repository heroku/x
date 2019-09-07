// Package testserver implements a simple gRPC server and client to use for
// testing. It will listen to a random port locally, and initialize the server
// and client ends.
package testserver

import (
	"log"

	"google.golang.org/grpc"

	"github.com/heroku/x/grpc/grpcserver"
)

// GRPCTestServer provides the Server and Conn (client).
type GRPCTestServer struct {
	Server   *grpc.Server
	Conn     *grpc.ClientConn
	localsrv *grpcserver.LocalServer
}

// New returns a new GRPCTestServer, configured to listen on a random local
// port.
func New(opts ...grpcserver.ServerOption) *GRPCTestServer {
	srv := grpcserver.New(opts...)

	return &GRPCTestServer{
		Server:   srv,
		localsrv: grpcserver.Local(srv),
	}
}

// Dial initiates a new gRPC connection to the server
// with the provided dial options.
func (t *GRPCTestServer) Dial(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return t.localsrv.Conn(opts...), nil
}

// Start will start the gRPC server in a goroutine.
func (t *GRPCTestServer) Start() error {
	go func() {
		if err := t.localsrv.Run(); err != nil {
			panic(err)
		}
	}()

	conn, err := t.Dial()
	if err != nil {
		return err
	}

	t.Conn = conn
	return nil
}

// Close closes the client connection, and stops the server from listening.
func (t *GRPCTestServer) Close() error {
	//TODO: SA1019: grpc.ErrClientConnClosing is deprecated: this error should not be relied upon by users; use the status code of Canceled instead.  (staticcheck)
	if err := t.Conn.Close(); err != nil && err != grpc.ErrClientConnClosing { //nolint:staticcheck
		log.Printf("GRPCTestServer failed to close client conn: %s", err)
	}

	t.localsrv.Stop(nil)
	return nil
}
