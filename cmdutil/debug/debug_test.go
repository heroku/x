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
		pprofConfig            *PProf
		expectedMemProfileRate int
	}{
		{
			name:         "test port as 9998 and mpf as 2",
			expectedAddr: "127.0.0.1:9998",
			pprofConfig: &PProf{
				Port:           9998,
				Enabled:        true,
				MemProfileRate: 524288,
			},
			expectedMemProfileRate: 524288,
		},
		{
			name:         "test port as 9997 and mpf as 4",
			expectedAddr: "127.0.0.1:9997",
			pprofConfig: &PProf{
				Port:           9997,
				Enabled:        true,
				MemProfileRate: 524287,
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

			// /debug/pprof/{profile,cmdline,symbol,trace} are handled by
			// separate funcs in net/http/pprof — Index alone returns 404
			// for them. Verify each is wired so CPU profiling works.
			extraPaths := []string{
				"/debug/pprof/profile?seconds=1",
				"/debug/pprof/cmdline",
				"/debug/pprof/symbol",
			}
			for _, p := range extraPaths {
				extURL := "http://" + server.addr + p
				t.Run("GET "+extURL, func(t *testing.T) {
					resp, err := client.Get(extURL)
					if err != nil {
						t.Fatalf("client.Get(%s) error = %v", extURL, err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						t.Errorf("status = %v, want %v", resp.StatusCode, http.StatusOK)
					}
				})
			}

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
