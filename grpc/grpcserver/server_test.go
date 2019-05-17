package grpcserver

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func ExampleLocal() {
	srv := New()
	localsrv := Local(srv)

	go localsrv.Run()
	defer localsrv.Stop(nil)

	c := healthpb.NewHealthClient(localsrv.Conn())

	resp, err := c.Check(context.Background(), &healthpb.HealthCheckRequest{}, grpc.FailFast(true))
	if err != nil {
		fmt.Printf("Error = %v", err)
		return
	}

	fmt.Printf("Status = %v", resp.Status)
	// Output: Status = SERVING
}
