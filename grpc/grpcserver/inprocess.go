package grpcserver

import (
	"net"
	"time"

	"github.com/hydrogen18/memlistener"
	"google.golang.org/grpc"
)

// Local returns an in-process server for the provided gRPC server.
func Local(s *grpc.Server) *LocalServer {
	return &LocalServer{
		ln:  memlistener.NewMemoryListener(),
		srv: s,
	}
}

// An LocalServer serves a gRPC server from memory.
type LocalServer struct {
	ln  *memlistener.MemoryListener
	srv *grpc.Server
}

// Run starts the in-process server.
//
// It implements oklog group's runFn.
func (s *LocalServer) Run() error {
	return s.srv.Serve(s.ln)
}

// Stop gracefully stops the gRPC server.
//
// It implements oklog group's interruptFn.
func (s *LocalServer) Stop(err error) {
	s.srv.GracefulStop()
}

// Conn returns a client connection to the in-process server.
func (s *LocalServer) Conn(opts ...grpc.DialOption) *grpc.ClientConn {
	defaultOptions := []grpc.DialOption{
		// TODO: SA1019: grpc.WithDialer is deprecated: use WithContextDialer instead  (staticcheck)
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) { //nolint:staticcheck
			return s.ln.Dial("mem", "")
		}),
		grpc.WithInsecure(),
	}

	conn, _ := grpc.Dial("", append(defaultOptions, opts...)...)
	return conn
}
