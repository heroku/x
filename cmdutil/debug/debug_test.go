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

	tests := []struct {
		name                   string
		expectedAddr           string
		pprofConfig            *PProfConfig
		expectedMemProfileRate int
	}{
		{
			name:         "test port as 9998 and mpf as 2",
			expectedAddr: "127.0.0.1:9998",
			pprofConfig: &PProfConfig{
				PProfPort:            9998,
				EnablePProfDebugging: true,
				MemProfileRate:       524288,
			},
			expectedMemProfileRate: 524288,
		},
		{
			name:         "test port as 9997 and mpf as 4",
			expectedAddr: "127.0.0.1:9997",
			pprofConfig: &PProfConfig{
				PProfPort:            9997,
				EnablePProfDebugging: true,
				MemProfileRate:       524287,
			},
			expectedMemProfileRate: 524287,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewPProfServer(logger, tt.pprofConfig)

			// Check server address
			if server.addr != tt.expectedAddr {
				t.Errorf("NewPProfServer() addr = %v, want %v", server.addr, tt.expectedAddr)
			}

			if runtime.MemProfileRate != tt.expectedMemProfileRate {
				t.Errorf("MemProfileRate expected  %v, got  %v", tt.expectedMemProfileRate, runtime.MemProfileRate)
			}

			// Start the server
			go func() {
				if err := server.Run(); err != nil {
					t.Errorf("NewPProfServer() run error = %v", err)
				}
			}()

			// Give the server a moment to start
			time.Sleep(100 * time.Millisecond)

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
