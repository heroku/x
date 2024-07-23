package grpcserver

import (
	"log/slog"
	"net"

	proxyproto "github.com/armon/go-proxyproto"
	"google.golang.org/grpc"
)

// TCP returns a TCP server for the provided gRPC server.
//
// The server transparently handles proxy protocol.
func TCP(l *slog.Logger, s *grpc.Server, addr string) *TCPServer {
	return &TCPServer{
		logger: l,
		srv:    s,
		addr:   addr,
	}
}

// A TCPServer serves a gRPC server over TCP with proxy-protocol support.
type TCPServer struct {
	logger *slog.Logger
	srv    *grpc.Server
	addr   string
}

// Run binds to the configured address and serves the gRPC server.
//
// It implements oklog group's runFn.
func (s *TCPServer) Run() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	proxyprotoLn := &proxyproto.Listener{Listener: ln}

	s.logger.With(
		slog.String("at", "binding"),
		slog.String("service", "grpc-tcp"),
		slog.String("addr", ln.Addr().String()),
	).Info("")

	return s.srv.Serve(proxyprotoLn)
}

// Stop gracefully stops the gRPC server.
//
// It implements oklog group's interruptFn.
func (s *TCPServer) Stop(err error) {
	s.srv.GracefulStop()
}
