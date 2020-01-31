package hmiddleware

import (
	"net/http"

	tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
)

// Tags provides a middleware function for seeding the context with a
// per-request tags that are eventually retrieved and logged by the
// StructuredLogger
func Tags(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		newTags := tags.NewTags()
		ctx = tags.SetInContext(ctx, newTags)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
