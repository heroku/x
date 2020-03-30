package hmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
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
		name                 string
		maxRequests          int
		requestCount         int
		wantCloseMetricCount float64
		wantCloseHeaders     int
	}{
		{
			name:         "too few requests",
			maxRequests:  10,
			requestCount: 5,
		},
		{
			name:                 "exactly max requests",
			maxRequests:          10,
			requestCount:         10,
			wantCloseMetricCount: 1,
			wantCloseHeaders:     1,
		},
		{
			name:                 "several times max requests",
			maxRequests:          10,
			requestCount:         100,
			wantCloseMetricCount: 10,
			wantCloseHeaders:     10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mp := testmetrics.NewProvider(t)
			ctx := ConnectionClosingContext(context.TODO(), nil)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("hello world"))
			})

			middleware := ConnectionClosingMiddleware(mp, test.maxRequests, 1024)
			handler := middleware(next)

			gotCloseHeader := 0
			for i := 0; i < test.requestCount; i++ {
				req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, req)

				if recorder.Header().Get("connection") == "close" {
					gotCloseHeader++
				}
			}

			mp.CheckCounter("server.connection.closes.total", test.wantCloseMetricCount)

			if want, got := test.wantCloseHeaders, gotCloseHeader; want != got {
				t.Fatalf("want close header count: %d, got %d", want, got)
			}
		})
	}
}

func BenchmarkConnectionClosingMiddleware_Cache1024(b *testing.B) {
	middleware := ConnectionClosingMiddleware(discard.New(), 100, 1024)
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
