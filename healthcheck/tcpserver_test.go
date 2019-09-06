package healthcheck

import (
	"io/ioutil"
	"net"
	"testing"
	"time"

	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestTCPServer(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	provider := testmetrics.NewProvider(t)
	server := NewTCPServer(logger, provider, "127.0.0.1:0")

	if err := server.start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		server.serve()
	}()

	conn, err := net.DialTimeout("tcp", server.ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("unable to dial server: %s", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second))

	data, err := ioutil.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(data), "OK\n"; got != want {
		t.Fatalf("response was %q, want %q", got, want)
	}

	// Assert server shuts down after stopping
	server.Stop(nil)
	<-done

	provider.CheckCounter("health", 1)
}
