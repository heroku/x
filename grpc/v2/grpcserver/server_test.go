package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/heroku/x/testing/mustcert"
)

func ExampleLocal() {
	srv := New()
	localsrv := Local(srv)

	go func() {
		if err := localsrv.Run(); err != nil {
			panic(err)
		}
	}()
	defer localsrv.Stop(nil)

	c := healthpb.NewHealthClient(localsrv.Conn())

	//TODO: SA1019: grpc.FailFast is deprecated: use WaitForReady.  (staticcheck)
	resp, err := c.Check(context.Background(), &healthpb.HealthCheckRequest{}, grpc.FailFast(true)) //nolint:staticcheck
	if err != nil {
		fmt.Printf("Error = %v", err)
		return
	}

	fmt.Printf("Status = %v", resp.Status)
	// Output: Status = SERVING
}

func TestGetPeerNameFromContext(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		if name := getPeerNameFromContext(context.Background()); name != "" {
			t.Errorf("name = %q want %q", name, "")
		}
	})

	t.Run("non-mTLS peer", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			// AuthInfo is nil if there is no transport security, based on the peer
			// package's docs.
			AuthInfo: nil,
		})

		if name := getPeerNameFromContext(ctx); name != "" {
			t.Errorf("name = %q want %q", name, "")
		}
	})

	t.Run("with an mTLS peer", func(t *testing.T) {
		clientName := "client"
		clientCert := mustcert.Leaf(clientName, nil)

		ctx := peer.NewContext(context.Background(), &peer.Peer{
			AuthInfo: credentials.TLSInfo{
				State: tls.ConnectionState{
					PeerCertificates: []*x509.Certificate{
						clientCert.TLS().Leaf,
					},
				},
			},
		})

		if name := getPeerNameFromContext(ctx); name != clientName {
			t.Errorf("name = %q want %q", name, clientName)
		}
	})
}

func TestValidatePeer(t *testing.T) {
	clientName := "client"
	clientCert := mustcert.Leaf(clientName, nil)
	validPeer := &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					clientCert.TLS().Leaf,
				},
			},
		},
	}

	t.Run("non-mTLS peer", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), &peer.Peer{
			// AuthInfo is nil if there is no transport security, based on the peer
			// package's docs.
			AuthInfo: nil,
		})

		err := validatePeer(ctx, func(*x509.Certificate) bool { return false })
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("err = %+v want %v", err, codes.Unauthenticated)
		}
	})

	t.Run("valid peer rejected by validator", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), validPeer)

		err := validatePeer(ctx, func(*x509.Certificate) bool { return false })
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("err = %+v want %v", err, codes.PermissionDenied)
		}
	})

	t.Run("valid peer accepted by validator", func(t *testing.T) {
		ctx := peer.NewContext(context.Background(), validPeer)

		err := validatePeer(ctx, func(*x509.Certificate) bool { return true })
		if err != nil {
			t.Fatalf("err = %+v want nil", err)
		}
	})
}
