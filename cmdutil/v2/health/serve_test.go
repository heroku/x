package health

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestTCPServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	srv := NewTCPServer(logger, meter, Config{Port: 0})

	// Run() returns error when Stop() closes listener - expected
	go func() {
		_ = srv.Run()
	}()

	time.Sleep(50 * time.Millisecond)

	tcpSrv := srv.(*tcpServer)
	addr := tcpSrv.ln.Addr().String()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 3)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(buf[:n]) != "OK\n" {
		t.Errorf("got %q, want %q", buf[:n], "OK\n")
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("no metrics collected")
	}

	var found bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "health" {
				found = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					if len(sum.DataPoints) > 0 && sum.DataPoints[0].Value > 0 {
						break
					}
				}
			}
		}
	}

	if !found {
		t.Error("health metric not incremented")
	}

	srv.Stop(nil)
}

func TestTickingServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	srv := NewTickingServer(logger, meter, Config{MetricInterval: 1})

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.Run()
	}()

	time.Sleep(2100 * time.Millisecond)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("no metrics collected")
	}

	var found bool
	var value int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "health" {
				found = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					if len(sum.DataPoints) > 0 {
						value = sum.DataPoints[0].Value
					}
				}
			}
		}
	}

	if !found {
		t.Fatal("health metric not found")
	}

	if value < 2 {
		t.Errorf("expected at least 2 ticks, got %d", value)
	}

	srv.Stop(nil)

	select {
	case <-ctx.Done():
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Errorf("Run() error = %v", err)
		}
	}
}
