package basicauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/route_guide/routeguide"

	"github.com/heroku/x/grpc/grpcclient"
	"github.com/heroku/x/grpc/grpcserver"
	"github.com/heroku/x/testing/testlog"
)

func TestGRPCPerRPCCredentialBasicAuth(t *testing.T) {
	l, _ := testlog.New()
	mux := http.NewServeMux()

	checker := NewChecker([]Credential{{"user", "pass"}})

	gSrv, hSrv := grpcserver.NewStandardH2C(
		mux,
		grpcserver.AuthInterceptors(
			grpc_auth.UnaryServerInterceptor(GRPCAuthFunc(checker)),
			grpc_auth.StreamServerInterceptor(GRPCAuthFunc(checker)),
		),
		grpcserver.LogEntry(l.WithField("at", "grpc")),
	)

	routeguide.RegisterRouteGuideServer(gSrv, &fakeServer{})

	srv := httptest.NewServer(hSrv.Handler)
	defer srv.Close()

	conn, err := grpcclient.DialH2CContext(
		context.Background(),
		srv.URL,
		grpc.WithPerRPCCredentials(&GRPCCredentials{Username: "user", Password: "pass"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := routeguide.NewRouteGuideClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.GetFeature(ctx, &routeguide.Point{})
	if err != nil {
		t.Fatal(err)
	}
}

type fakeServer struct {
	routeguide.UnimplementedRouteGuideServer
}

func (s *fakeServer) GetFeature(context.Context, *routeguide.Point) (*routeguide.Feature, error) {
	return &routeguide.Feature{}, nil
}
