package hmiddleware

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/heroku/x/go-kit/metrics/provider/discard"
	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestConnectionClosingContext(t *testing.T) {
	ctx := ConnectionClosingContext(context.TODO(), nil)
	connID := ctx.Value(connCloseID).(string)
	if connID == "" {
		t.Fatal("want connection id in context, got nil")
	}
}

func TestConnectionClosingMiddleware(t *testing.T) {
	tests := []struct {
		name             string
		maxRequests      int
		requestCount     int
		wantConnections  int
		wantCloseHeaders int
	}{
		{
			name:            "too few requests",
			maxRequests:     10,
			requestCount:    5,
			wantConnections: 1,
		},
		{
			name:             "exactly max requests",
			maxRequests:      10,
			requestCount:     10,
			wantConnections:  1,
			wantCloseHeaders: 1,
		},
		{
			name:             "several times max requests",
			maxRequests:      10,
			requestCount:     100,
			wantConnections:  10,
			wantCloseHeaders: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				gotConnIDs = make(map[string]bool)
				mtx        sync.Mutex
			)
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connIDValue := r.Context().Value(connCloseID)
				if connID, ok := connIDValue.(string); ok {
					mtx.Lock()
					gotConnIDs[connID] = true
					mtx.Unlock()
				}
				_, _ = w.Write([]byte("hello world"))
			})

			mp := testmetrics.NewProvider(t)
			middleware, err := ConnectionClosingMiddleware(mp, test.maxRequests, 1024)
			if err != nil {
				t.Fatal(err)
			}

			handler = middleware(handler)

			server := httptest.NewUnstartedServer(handler)
			server.Config.ConnContext = ConnectionClosingContext //nolint:typecheck
			server.Start()
			defer server.Close()

			client := server.Client()

			gotCloseHeader := 0
			for i := 0; i < test.requestCount; i++ {
				req, err := http.NewRequest(http.MethodGet, server.URL, nil)
				if err != nil {
					t.Fatal(err)
				}

				res, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.Copy(ioutil.Discard, res.Body)
				_ = res.Body.Close()

				if res.Close {
					gotCloseHeader++
				}
			}

			if want, got := test.wantConnections, len(gotConnIDs); want != got {
				t.Fatalf("want unique connects: %d, got %d", want, got)
			}

			mp.CheckCounter("connection.closes.total", float64(test.wantCloseHeaders))

			if want, got := test.wantCloseHeaders, gotCloseHeader; want != got {
				t.Fatalf("want close header count: %d, got %d", want, got)
			}
		})
	}
}

func BenchmarkConnectionClosingMiddleware_Cache1024(b *testing.B) {
	middleware, err := ConnectionClosingMiddleware(discard.New(), 100, 1024)
	if err != nil {
		b.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})
	handler := middleware(next)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := ConnectionClosingContext(context.TODO(), nil)
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}
}
