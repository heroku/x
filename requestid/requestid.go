package requestid

import "net/http"

var requestIDKeys = []string{
	"Request-ID", "X-Request-ID",
}

// Get reads the Request-ID and X-Request-ID HTTP header from an `*http.Request`
// If no header is set, an empty string is returned
func Get(r *http.Request) string {
	for _, try := range requestIDKeys {
		if id := r.Header.Get(try); id != "" {
			return id
		}
	}
	return ""
}
