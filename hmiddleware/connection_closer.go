package hmiddleware

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/heroku/x/go-kit/metrics"
)

type connCloseKey int

var connCloseID connCloseKey

// ConnectionClosingContext adds a unique identifier to the context for a newly established connection. This function
// is meant to be used as an http.Server's ConnContext field, and is required for the ConnectionClosingMiddleware.
func ConnectionClosingContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, connCloseID, uuid.New().String())
}

// ConnectionClosingMiddleware returns middleware that will add a "Connection: close" responses header after a single
// connection has produced maxRequests http requests. The cacheSize determines the size of the LRU cache which tracks
// connection request counts. This middleware requires the http.Server sets ConnectionClosingContext for it's
// ConnContext field.
func ConnectionClosingMiddleware(mp metrics.Provider, maxRequests, cacheSize int) (func(http.Handler) http.Handler, error) {
	var (
		closed = mp.NewCounter("connection.closes.total")
		mtx    sync.Mutex
	)

	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	handler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			shouldClose := false
			connID, ok := r.Context().Value(connCloseID).(string)

			if ok {
				// In the case of http/1.x requests are served serially per connection, and locking isn't necessary.
				// However this isn't necessarily true for the http/2 path. So we lock here for correctness.
				mtx.Lock()

				requests := 0
				requestValue, ok := cache.Get(connID)
				if ok {
					requests = requestValue.(int)
				}

				requests++
				if requests >= maxRequests {
					shouldClose = true
					cache.Remove(connID)
				} else {
					cache.Add(connID, requests)
				}

				mtx.Unlock()
			}

			if shouldClose {
				closed.Add(1)
				w.Header().Add("connection", "close")
			}

			next.ServeHTTP(w, r)
		})
	}

	return handler, nil
}
