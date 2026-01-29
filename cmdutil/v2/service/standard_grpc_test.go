package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type testServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (s *testServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *testServer) Start(srv *grpc.Server) error {
	grpc_health_v1.RegisterHealthServer(srv, s)
	return nil
}

func TestGRPCHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	grpcHandler := GRPCHandler(logger, &testServer{})

	srv := httptest.NewUnstartedServer(grpcHandler)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	conn, err := grpc.NewClient(srv.Listener.Addr().String(), 
		grpc.WithTransportCredentials(credentials.NewTLS(srv.Client().Transport.(*http.Transport).TLSClientConfig)))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("got status %v, want SERVING", resp.Status)
	}
}

func TestWithGRPC(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("HTTP OK"))
	})

	combined := WithGRPC(httpHandler, logger, &testServer{})

	srv := httptest.NewUnstartedServer(combined)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	// Test gRPC
	conn, err := grpc.NewClient(srv.Listener.Addr().String(), 
		grpc.WithTransportCredentials(credentials.NewTLS(srv.Client().Transport.(*http.Transport).TLSClientConfig)))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("got status %v, want SERVING", resp.Status)
	}

	// Test HTTP
	httpResp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)
	if string(body) != "HTTP OK" {
		t.Errorf("got %q, want %q", body, "HTTP OK")
	}
}
