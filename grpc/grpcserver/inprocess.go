package grpcserver

import (
	"context"
	"net"

	"github.com/hydrogen18/memlistener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
func (s *LocalServer) Stop(error) {
	s.srv.GracefulStop()
}

// Conn returns a client connection to the in-process server.
func (s *LocalServer) Conn(opts ...grpc.DialOption) *grpc.ClientConn {
	defaultOptions := []grpc.DialOption{
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.ln.Dial("mem", "")
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, _ := grpc.NewClient("passthrough:///", append(defaultOptions, opts...)...)

	conn.Connect()
	return conn
}
