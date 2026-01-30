package service

import (
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc"
)

// GRPCStarter is implemented by gRPC services that can register themselves with a grpc.Server.
type GRPCStarter interface {
	Start(srv *grpc.Server) error
}

// GRPCHandler returns an http.Handler that serves gRPC requests.
// Non-gRPC requests receive 404.
func GRPCHandler(l *slog.Logger, server GRPCStarter) http.Handler {
	srv := grpc.NewServer()
	if err := server.Start(srv); err != nil {
		l.Error("failed to start grpc server", slog.Any("error", err))
		return http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			srv.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
}

// WithGRPC wraps an HTTP handler to also serve gRPC requests on the same port.
// gRPC requests (HTTP/2 with application/grpc content-type) are handled by the
// gRPC server, all other requests go to the HTTP handler.
func WithGRPC(httpHandler http.Handler, l *slog.Logger, server GRPCStarter) http.Handler {
	srv := grpc.NewServer()
	if err := server.Start(srv); err != nil {
		l.Error("failed to start grpc server", slog.Any("error", err))
		return httpHandler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			srv.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}
