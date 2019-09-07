package grpc

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/heroku/x/grpc/grpcclient"
	"github.com/heroku/x/grpc/grpcserver"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// TestH2CE2E exercises both the H2C server and H2C client in an end to end
// fashion
func TestH2CE2E(t *testing.T) {
	handle11resp := "http 1.1 requested"
	handle11 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, handle11resp)
	})

	gSrv, hSrv := grpcserver.NewStandardH2C(handle11)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Error starting HTTP listener [%+v]", err)
	}

	go func() {
		if err := hSrv.Serve(lis); err != nil && err != http.ErrServerClosed {
			panic("unexpected error: " + err.Error())
		}
	}()
	defer func() {
		if err := hSrv.Shutdown(context.TODO()); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	defer gSrv.GracefulStop()

	tr := &http.Transport{}
	defer tr.CloseIdleConnections()

	cl := &http.Client{Transport: tr}

	resp, err := cl.Get("http://" + lis.Addr().String())
	if err != nil {
		t.Errorf("Error making HTTP/1.1 call to H2C Server [%+v]", err)
	}
	defer resp.Body.Close()

	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading HTTP/1.1 call to H2C Server body [%+v]", err)
	}

	if string(bb) != handle11resp {
		t.Errorf("Expected HTTP/1.1 call to return %q, got %q", handle11resp, string(bb))
	}

	//TODO: SA1019: grpc.WithWaitForHandshake is deprecated: this is the default behavior, and this option will be removed after the 1.18 release.  (staticcheck)
	conn, err := grpcclient.DialH2C("http://"+lis.Addr().String(), grpc.WithWaitForHandshake()) //nolint:staticcheck
	if err != nil {
		t.Fatalf("Error dialing server [%+v]", err)
	}
	defer conn.Close()

	hc := healthpb.NewHealthClient(conn)

	// ignore the response, we only care than the transport works
	_, err = hc.Check(
		context.Background(),
		&healthpb.HealthCheckRequest{},
		//TODO: SA1019: grpc.FailFast is deprecated: use WaitForReady.  (staticcheck)
		grpc.FailFast(false), //nolint:staticcheck
	)
	if err != nil {
		t.Errorf("Error calling health backend [%+v]", err)
	}
}
