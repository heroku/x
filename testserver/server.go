// Package testserver implements a simple gRPC server and client to use for
// testing. It will listen to a random port locally, and initialize the server
// and client ends.
package testserver

import (
	"log"
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

// Dial initiates a new gRPC connection to the server
// with the provided dial options.
func (t *GRPCTestServer) Dial(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	defaultOptions := []grpc.DialOption{
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return t.listener.Dial("", "")
		}),
		grpc.WithInsecure(),
	}

	return grpc.Dial("", append(defaultOptions, opts...)...)
}

// Start will start the gRPC server in a goroutine.
func (t *GRPCTestServer) Start() error {
	go t.Server.Serve(t.listener)

	conn, err := t.Dial()
	if err != nil {
		return err
	}

	t.Conn = conn
	return nil
}

// Close closes the client connection, and stops the server from listening.
func (t *GRPCTestServer) Close() error {
	if err := t.Conn.Close(); err != nil && err != grpc.ErrClientConnClosing {
		log.Printf("GRPCTestServer failed to close client conn: %s", err)
	}

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			log.Println("GRPCTestServer failed to stop gracefully, stopping now")
			t.Server.Stop()
		}
	}()

	t.Server.GracefulStop()
	return nil
}
