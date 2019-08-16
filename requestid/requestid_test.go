package requestid

import (
	"net/http"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name:    "send Request-ID",
			headers: map[string]string{"Request-ID": "request-id-value"},
			want:    "request-id-value",
		},
		{
			name:    "send X-Request-ID",
			headers: map[string]string{"X-Request-ID": "x-request-id-value"},
			want:    "x-request-id-value",
		},
		{
			name: "send Request-ID and X-Request-ID. Prioritize Request-ID",
			headers: map[string]string{
				"Request-ID":   "request-id-value",
				"X-Request-ID": "x-request-id-value",
			},
			want: "request-id-value",
		},
		{
			name: "send invalid request id header",
			headers: map[string]string{
				"request_id": "request_id_value",
			},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "http://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}

			for k, v := range test.headers {
				r.Header.Set(k, v)
			}

			got := Get(r)
			if got != test.want {
				t.Fatalf("unexpected request-id got %q; want %q", got, test.want)
			}

		})
	}

}
