package grpchealthcheck

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

type response struct {
	success bool
}

type testStream struct {
	ctx     context.Context
	recvMsg func(interface{}) error
	grpc.ClientStream
}

func (s *testStream) Context() context.Context {
	return s.ctx
}

func (s *testStream) RecvMsg(m interface{}) error {
	return s.recvMsg(m)
}

type testHealthWatchClient struct {
}

func (hwc *testHealthWatchClient) Recv() (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}
func (hwc *testHealthWatchClient) RecvMsg(m interface{}) error {
	return nil
}

func (hwc *testHealthWatchClient) SendMsg(m interface{}) error {
	return nil
}

func (hwc *testHealthWatchClient) CloseSend() error {
	return nil
}

func (hwc *testHealthWatchClient) Context() context.Context {
	return context.TODO()
}

func (hwc *testHealthWatchClient) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (hwc *testHealthWatchClient) Trailer() metadata.MD {
	return metadata.MD{}
}

type testHealthClient struct {
	unhealthy bool
	checks    int
	sync.Mutex
}

func (c *testHealthClient) NumChecks() int {
	c.Lock()
	defer c.Unlock()

	return c.checks
}

func (c *testHealthClient) Check(ctx context.Context, in *healthpb.HealthCheckRequest, opts ...grpc.CallOption) (*healthpb.HealthCheckResponse, error) {
	c.Lock()
	defer c.Unlock()

	c.checks++

	status := healthpb.HealthCheckResponse_SERVING
	if c.unhealthy {
		status = healthpb.HealthCheckResponse_NOT_SERVING
	}

	return &healthpb.HealthCheckResponse{Status: status}, nil
}

func (c *testHealthClient) Watch(ctx context.Context, in *healthpb.HealthCheckRequest, opts ...grpc.CallOption) (healthpb.Health_WatchClient, error) {
	return &testHealthWatchClient{}, nil
}

func TestStreamInterceptor(t *testing.T) {
	stream := &testStream{
		ctx: context.Background(),
		recvMsg: func(m interface{}) error {
			r := m.(*response)
			r.success = true
			return nil
		},
	}

	healthStream := &healthCheckStream{
		interval:     time.Minute,
		health:       &testHealthClient{},
		ClientStream: stream,
	}

	var out response
	if err := healthStream.RecvMsg(&out); err != nil {
		t.Fatal(err)
	}

	if !out.success {
		t.Fatalf("success = %v, want %v", out.success, true)
	}
}

func TestStreamInterceptorDelayedResponse(t *testing.T) {
	var (
		interval = 5 * time.Millisecond
		delay    = interval * 4
	)

	stream := &testStream{
		ctx: context.Background(),
		recvMsg: func(m interface{}) error {
			time.Sleep(delay)

			r := m.(*response)
			r.success = true
			return nil
		},
	}

	health := &testHealthClient{}

	healthStream := &healthCheckStream{
		interval:     interval,
		health:       health,
		ClientStream: stream,
	}

	var out response
	if err := healthStream.RecvMsg(&out); err != nil {
		t.Fatal(err)
	}

	if !out.success {
		t.Fatalf("success = %v, want %v", out.success, true)
	}

	if checks := health.NumChecks(); checks <= 0 {
		t.Fatalf("received %d health checks, want >= 1", checks)
	}
}

func TestStreamInterceptorServerUnhealthy(t *testing.T) {
	var (
		interval = 5 * time.Millisecond
		delay    = interval * 4
	)

	stream := &testStream{
		ctx: context.Background(),
		recvMsg: func(m interface{}) error {
			time.Sleep(delay)

			r := m.(*response)
			r.success = true
			return nil
		},
	}

	health := &testHealthClient{unhealthy: true}

	healthStream := &healthCheckStream{
		interval:     interval,
		health:       health,
		ClientStream: stream,
	}

	var out response
	if err := healthStream.RecvMsg(&out); err == nil {
		t.Fatal("RecvMsg succeeded, want error")
	}
}
