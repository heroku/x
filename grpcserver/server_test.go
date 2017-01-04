package grpcserver

import (
	"testing"

	"github.com/heroku/cedar/lib/grpc/grpcclient"
)

// NewInProcess creates a new grpc.Server, starts it and registers it in the
// connection registry using the given service name.
func TestNewInProcess(t *testing.T) {
	name := "test-server"
	ts, err := NewInProcess(name)
	if err != nil {
		t.Fatal(err)
	}
	defer ts.Close()
	defer grpcclient.DeregisterConnection(name)
	s := grpcclient.Conn(name)
	if ts.Conn != s {
		t.Fatalf("got %v but want %v", s, ts.Conn)
	}
}
