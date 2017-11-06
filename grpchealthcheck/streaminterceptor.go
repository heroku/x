package grpchealthcheck

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewStreamInterceptor returns a gRPC StreamClientInterceptor
// which performs server health checks on the given interval
// for any stream which is initiated.
//
//		health := grpchealthcheck.NewStreamInterceptor(30 * time.Second)
//		conn, _ := grpc.Dial("server", grpc.WithStreamInterceptor(health))
//
func NewStreamInterceptor(interval time.Duration) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		return &healthCheckStream{
			health:       healthpb.NewHealthClient(cc),
			interval:     interval,
			ClientStream: stream,
		}, nil
	}
}

type healthCheckStream struct {
	health   healthpb.HealthClient
	interval time.Duration

	grpc.ClientStream
}

func (s *healthCheckStream) RecvMsg(m interface{}) error {
	healthcheck := time.NewTimer(s.interval)
	defer healthcheck.Stop()

	result := make(chan error, 1)
	go func() { result <- s.ClientStream.RecvMsg(m) }()

	for {
		select {
		case err := <-result:
			return err
		case <-healthcheck.C:
		}

		if err := s.healthcheck(); err != nil {
			return err
		}
		healthcheck.Reset(s.interval)
	}
}

func (s *healthCheckStream) healthcheck() error {
	r, err := s.health.Check(s.Context(), &healthpb.HealthCheckRequest{})
	if err != nil {
		return err
	}

	if r.Status != healthpb.HealthCheckResponse_SERVING {
		return errors.Errorf("server status = %v, want %v", r.Status, healthpb.HealthCheckResponse_SERVING)
	}

	return nil
}
