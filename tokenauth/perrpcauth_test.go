package tokenauth

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/examples/route_guide/routeguide"
)

type fakeAuthorizer struct {
	err error
}

func (i *fakeAuthorizer) Authorize(ctx context.Context, req *RPC, creds map[string]string) (context.Context, error) {
	_, okuser := creds["username"]
	_, okpass := creds["password"]
	if !okuser || !okpass {
		return ctx, grpc.Errorf(codes.Unauthenticated, "Invalid creds passed to authorizer")
	}
	return ctx, i.err
}

type routeGuide struct {
	routeguide.RouteGuideServer
}

func (r *routeGuide) GetFeature(_ context.Context, _ *routeguide.Point) (*routeguide.Feature, error) {
	return &routeguide.Feature{}, nil
}

func (r *routeGuide) ListFeatures(_ *routeguide.Rectangle, svr routeguide.RouteGuide_ListFeaturesServer) error {
	if err := svr.Send(&routeguide.Feature{}); err != nil {
		panic(err)
	}
	return nil
}

func TestGRPCServerInterceptor(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	authorizer := &fakeAuthorizer{}
	server := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryServerInterceptor(authorizer)),
		grpc.StreamInterceptor(StreamServerInterceptor(authorizer)),
	)
	defer server.GracefulStop()
	routeguide.RegisterRouteGuideServer(server, &routeGuide{})
	go server.Serve(listener)

	for _, tc := range []struct {
		name    string
		creds   map[string]string
		authErr error
		wantErr error
	}{
		{
			"Valid request",
			map[string]string{"username": "user", "password": "pass"},
			nil,
			nil,
		},
		{
			"Empty credentials",
			map[string]string{},
			nil,
			errors.New("Invalid creds passed to authorizer"),
		},
		{
			"Authorizer returns error",
			map[string]string{"username": "user", "password": "pass"},
			errors.New("authorization failed"),
			errors.New("authorization failed"),
		},
	} {
		rc := &RPCCredentials{
			Credentials:   tc.creds,
			AllowInsecure: true,
		}
		conn, err := grpc.Dial(
			listener.Addr().String(),
			grpc.WithInsecure(),
			grpc.WithBlock(),
			grpc.WithPerRPCCredentials(rc),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		authorizer.err = tc.authErr
		rg := routeguide.NewRouteGuideClient(conn)

		_, err = rg.GetFeature(context.TODO(), &routeguide.Point{})
		if grpc.ErrorDesc(err) != grpc.ErrorDesc(tc.wantErr) {
			t.Errorf("[%v Unary] want err %+v, got %+v", tc.name, tc.wantErr, err)
		}

		cl, err := rg.ListFeatures(context.TODO(), &routeguide.Rectangle{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = cl.Recv()
		if grpc.ErrorDesc(err) != grpc.ErrorDesc(tc.wantErr) {
			t.Errorf("[%v Stream] want err %+v, got %+v", tc.name, tc.wantErr, err)
		}
	}
}
