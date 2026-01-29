package metrics

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"google.golang.org/grpc"

	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
)

func TestSetupDisabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	provider, shutdown, err := Setup(context.Background(), cfg, "test-service", "test-namespace", "test", "instance-1")
	if err != nil {
		t.Fatalf("Setup with disabled config should not error: %v", err)
	}
	if provider == nil {
		t.Fatal("Setup should return non-nil provider even when disabled")
	}
	if shutdown == nil {
		t.Fatal("Setup should return non-nil shutdown function")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown should not error: %v", err)
	}
}

func TestSetupRequiresEndpoint(t *testing.T) {
	cfg := Config{
		Enabled: true,
	}

	_, _, err := Setup(context.Background(), cfg, "test-service", "test-namespace", "test", "instance-1")
	if err == nil {
		t.Fatal("Setup should error when endpoint is nil")
	}
}

func TestSetupUnsupportedProtocol(t *testing.T) {
	endpoint, _ := url.Parse("http://localhost:4318")
	cfg := Config{
		Enabled:  true,
		Endpoint: endpoint,
		Protocol: "invalid",
	}

	_, _, err := Setup(context.Background(), cfg, "test-service", "test-namespace", "test", "instance-1")
	if err == nil {
		t.Fatal("Setup should error with unsupported protocol")
	}
}

func TestSetupHTTPInsecure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoint, _ := url.Parse(server.URL)
	cfg := Config{
		Enabled:  true,
		Endpoint: endpoint,
		Protocol: "http/protobuf",
		Interval: time.Minute,
	}

	provider, shutdown, err := Setup(context.Background(), cfg, "test-service", "test-namespace", "production", "instance-1")
	if err != nil {
		t.Fatalf("Setup should not error: %v", err)
	}
	if provider == nil {
		t.Fatal("Setup should return non-nil provider")
	}
	if shutdown == nil {
		t.Fatal("Setup should return non-nil shutdown function")
	}

	// Verify we can create a meter
	meter := provider.Meter("test")
	if meter == nil {
		t.Fatal("Provider should return non-nil meter")
	}

	// Verify we can create a counter
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}
	if counter == nil {
		t.Fatal("Meter should return non-nil counter")
	}

	// Record a value
	counter.Add(context.Background(), 1)

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown should not error: %v", err)
	}
}

func TestSetupHTTP(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoint, _ := url.Parse(server.URL)
	cfg := Config{
		Enabled:  true,
		Endpoint: endpoint,
		Protocol: "http/protobuf",
		Interval: time.Minute,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	provider, shutdown, err := Setup(context.Background(), cfg, "test-service", "test-namespace", "production", "instance-1", WithTLSConfig(tlsConfig))
	if err != nil {
		t.Fatalf("Setup should not error: %v", err)
	}
	if provider == nil {
		t.Fatal("Setup should return non-nil provider")
	}

	meter := provider.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}
	counter.Add(context.Background(), 1)

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown should not error: %v", err)
	}
}

// mockMetricsService implements the OTLP metrics service
type mockMetricsService struct {
	collectormetrics.UnimplementedMetricsServiceServer
}

func (m *mockMetricsService) Export(ctx context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	return &collectormetrics.ExportMetricsServiceResponse{}, nil
}

func TestSetupGRPC(t *testing.T) {
	grpcServer := grpc.NewServer()
	collectormetrics.RegisterMetricsServiceServer(grpcServer, &mockMetricsService{})

	server := httptest.NewUnstartedServer(grpcServer)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	defer grpcServer.Stop()

	endpoint, _ := url.Parse(server.URL)
	cfg := Config{
		Enabled:  true,
		Endpoint: endpoint,
		Protocol: "grpc",
		Interval: time.Minute,
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	provider, shutdown, err := Setup(context.Background(), cfg, "test-service", "test-namespace", "production", "instance-1", WithTLSConfig(tlsConfig))
	if err != nil {
		t.Fatalf("Setup should not error: %v", err)
	}
	if provider == nil {
		t.Fatal("Setup should return non-nil provider")
	}

	meter := provider.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}
	counter.Add(context.Background(), 1)

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown should not error: %v", err)
	}
}
