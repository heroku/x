package hmiddleware

import (
	"net/http"

	"github.com/heroku/x/hcontext"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID, _ := hcontext.FromRequest(r)
		ctx = hcontext.WithRequestID(ctx, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
