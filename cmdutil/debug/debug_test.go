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
		name                  string
		config                PProfServerConfig
		expectedAddr          string
		expectedMutexFraction int
	}{
		{
			name:                  "DefaultAddr",
			config:                PProfServerConfig{},
			expectedAddr:          "127.0.0.1:9998",
			expectedMutexFraction: defaultMutexProfileFraction,
		},
		{
			name:                  "CustomAddr",
			config:                PProfServerConfig{Addr: "127.0.0.1:9090"},
			expectedAddr:          "127.0.0.1:9090",
			expectedMutexFraction: defaultMutexProfileFraction,
		},
		{
			name:                  "CustomMutexProfileFraction",
			config:                PProfServerConfig{MutexProfileFraction: 5},
			expectedAddr:          "127.0.0.1:9998",
			expectedMutexFraction: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewPProfServer(tt.config, logger)

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

			// Perform HTTP GET requests to ensure the server is running and all handlers respond correctly
			profiles := []string{"", "heap", "goroutine", "threadcreate", "block", "mutex"}
			for _, profile := range profiles {
				url := "http://" + server.addr + "/debug/pprof/" + profile
				resp, err := http.Get(url)
				if err != nil {
					t.Errorf("http.Get(%s) error = %v", url, err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("http.Get(%s) status = %v, want %v", url, resp.StatusCode, http.StatusOK)
				}
			}

			// urls := []string{
			// 	"http://" + server.addr + "/debug/pprof/",
			// 	"http://" + server.addr + "/debug/pprof/heap",
			// 	"http://" + server.addr + "/debug/pprof/goroutine",
			// 	"http://" + server.addr + "/debug/pprof/threadcreate",
			// 	"http://" + server.addr + "/debug/pprof/block",
			// 	"http://" + server.addr + "/debug/pprof/mutex",
			// }

			// // Perform HTTP GET requests to ensure the server is running and all handlers respond correctly
			// client := &http.Client{}
			// for _, url := range urls {
			// 	t.Run("GET "+url, func(t *testing.T) {
			// 		req, err := http.NewRequest("GET", url, nil)
			// 		if err != nil {
			// 			t.Errorf("http.NewRequest(%s) error = %v", url, err)
			// 		}

			// 		resp, err := client.Do(req)
			// 		if err != nil {
			// 			t.Errorf("http.Client.Do() error = %v", err)
			// 		}

			// 		if resp.StatusCode != http.StatusOK {
			// 			t.Errorf("http.Client.Do() status = %v, want %v", resp.StatusCode, http.StatusOK)
			// 		}

			// 		resp.Body.Close()
			// 	})
			// }

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
