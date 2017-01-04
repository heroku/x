// Package testserver implements a simple gRPC server and client to use for
// testing. It will listen to a random port locally, and initialize the server
// and client ends.
package testserver

import (
	"net"
	"time"

	"github.com/hydrogen18/memlistener"
	"google.golang.org/grpc"
)

// GRPCTestServer provides the Server and Conn (client).
type GRPCTestServer struct {
	Server   *grpc.Server
	Conn     *grpc.ClientConn
	listener *memlistener.MemoryListener
}

// New returns a new GRPCTestServer, configured to listen on a random local
// port.
func New() (*GRPCTestServer, error) {
	return &GRPCTestServer{
		Server:   grpc.NewServer(),
		listener: memlistener.NewMemoryListener(),
	}, nil
}

// Start will start the gRPC server in a goroutine.
func (t *GRPCTestServer) Start() error {
	go t.Server.Serve(t.listener)

	conn, err := grpc.Dial("",
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return t.listener.Dial("", "")
		}),
		grpc.WithInsecure(),
	)
	t.Conn = conn
	return err
}

// Close closes the client connection, and stops the server from listening.
func (t *GRPCTestServer) Close() error {
	if err := t.Conn.Close(); err != nil {
		return err
	}
	t.Server.GracefulStop()
	return nil
}
