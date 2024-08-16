package debug

import (
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewPProfServer(t *testing.T) {
	logger := logrus.New()
	defaultMutexProfileFraction := 2

	tests := []struct {
		name                  string
		expectedAddr          string
		expectedMutexFraction int
		port                  int
		mutexProfileFraction  int
	}{
		{
			name:                  "test port as 9998 and mpf as 2",
			expectedAddr:          "127.0.0.1:9998",
			port:                  9998,
			mutexProfileFraction:  defaultMutexProfileFraction,
			expectedMutexFraction: defaultMutexProfileFraction,
		},
		{
			name:                  "test port as 9997 and mpf as 4",
			expectedAddr:          "127.0.0.1:9997",
			port:                  9997,
			mutexProfileFraction:  4,
			expectedMutexFraction: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewPProfServer(logger, tt.port, tt.mutexProfileFraction)

			// Check server address
			if server.addr != tt.expectedAddr {
				t.Errorf("NewPProfServer() addr = %v, want %v", server.addr, tt.expectedAddr)
			}

			// Start the server
			go func() {
				if err := server.Run(); err != nil {
					t.Errorf("NewPProfServer() run error = %v", err)
				}
			}()

			// Give the server a moment to start
			time.Sleep(100 * time.Millisecond)

			// Check mutex profile fraction
			if got := runtime.SetMutexProfileFraction(0); got != tt.expectedMutexFraction {
				t.Errorf("runtime.SetMutexProfileFraction() = %v, want %v", got, tt.expectedMutexFraction)
			}
			runtime.SetMutexProfileFraction(tt.expectedMutexFraction) // Reset to the expected value

			// Perform HTTP GET request to the root path
			url := "http://" + server.addr + "/debug/pprof/"
			client := &http.Client{}

			t.Run("GET "+url, func(t *testing.T) {
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					t.Errorf("http.NewRequest(%s) error = %v", url, err)
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("http.Client.Do() error = %v", err)
				}

				if resp.StatusCode != http.StatusOK {
					t.Errorf("http.Client.Do() status = %v, want %v", resp.StatusCode, http.StatusOK)
				}

				resp.Body.Close()
			})

			// Stop the server
			server.Stop(nil)

			// Ensure the server is stopped
			select {
			case <-server.done:
				// success
			case <-time.After(1 * time.Second):
				t.Fatal("server did not stop in time")
			}
		})
	}
}
