package grpcserver

import (
	"net"

	proxyproto "github.com/armon/go-proxyproto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

// TCP returns a TCP server for the provided gRPC server.
//
// The server transparently handles proxy protocol.
func TCP(l zerolog.Logger, s *grpc.Server, addr string) *TCPServer {
	return &TCPServer{
		logger: l,
		srv:    s,
		addr:   addr,
	}
}

// A TCPServer serves a gRPC server over TCP with proxy-protocol support.
type TCPServer struct {
	logger zerolog.Logger
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

	s.logger.Info().
		Str("at", "binding").
		Str("service", "grpc-tcp").
		Str("addr", ln.Addr().String()).
		Send()

	return s.srv.Serve(proxyprotoLn)
}

// Stop gracefully stops the gRPC server.
//
// It implements oklog group's interruptFn.
func (s *TCPServer) Stop(err error) {
	s.srv.GracefulStop()
}
