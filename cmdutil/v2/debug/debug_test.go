package debug

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/pprof/profile"
)

func TestNewPProfServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

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

			if server.addr != tt.expectedAddr {
				t.Errorf("NewPProfServer() addr = %v, want %v", server.addr, tt.expectedAddr)
			}

			if runtime.MemProfileRate != tt.expectedMemProfileRate {
				t.Errorf("MemProfileRate expected  %v, got  %v", tt.expectedMemProfileRate, runtime.MemProfileRate)
			}

			go func() {
				if err := server.Run(); err != nil {
					t.Errorf("NewPProfServer() run error = %v", err)
				}
			}()

			time.Sleep(100 * time.Millisecond)

			client := &http.Client{}

			t.Run("GET index", func(t *testing.T) {
				resp, err := client.Get("http://" + server.addr + "/debug/pprof/")
				if err != nil {
					t.Fatalf("GET index failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("status = %v, want %v", resp.StatusCode, http.StatusOK)
				}

				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), "Types of profiles available") {
					t.Error("index page missing expected pprof content")
				}
			})

			t.Run("GET heap profile", func(t *testing.T) {
				resp, err := client.Get("http://" + server.addr + "/debug/pprof/heap")
				if err != nil {
					t.Fatalf("GET heap failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("status = %v, want %v", resp.StatusCode, http.StatusOK)
				}

				body, _ := io.ReadAll(resp.Body)

				// Verify it's gzipped
				if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
					t.Error("heap profile not gzipped")
				}

				// Parse and inspect the heap profile
				gz, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					t.Fatalf("failed to decompress: %v", err)
				}
				defer gz.Close()

				p, err := profile.Parse(gz)
				if err != nil {
					t.Fatalf("invalid pprof format: %v", err)
				}

				// Heap profile contains memory allocation data
				t.Logf("Heap profile: %d samples, %d locations", len(p.Sample), len(p.Location))
				for _, st := range p.SampleType {
					t.Logf("  Sample type: %s/%s", st.Type, st.Unit)
				}
				
				// Heap profiles have sample types like:
				// - alloc_objects/count (number of allocations)
				// - alloc_space/bytes (bytes allocated)
				// - inuse_objects/count (currently allocated objects)
				// - inuse_space/bytes (currently allocated bytes)
				if len(p.SampleType) == 0 {
					t.Error("heap profile has no sample types")
				}
			})

			t.Run("GET cpu profile", func(t *testing.T) {
				// Start CPU-intensive work to generate samples
				done := make(chan struct{})
				go func() {
					data := []byte("benchmark data for cpu profiling")
					for {
						select {
						case <-done:
							return
						default:
							// CPU-intensive crypto work
							hash := sha256.Sum256(data)
							data = hash[:]
						}
					}
				}()

				resp, err := client.Get("http://" + server.addr + "/debug/pprof/profile?seconds=1")
				close(done)
				
				if err != nil {
					t.Fatalf("GET profile failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("cpu profile status = %v, want %v", resp.StatusCode, http.StatusOK)
				}

				body, _ := io.ReadAll(resp.Body)

				// Verify it's gzipped
				if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
					t.Error("cpu profile not gzipped")
				}

				// Parse and inspect the CPU profile
				gz, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					t.Fatalf("failed to decompress: %v", err)
				}
				defer gz.Close()

				p, err := profile.Parse(gz)
				if err != nil {
					t.Fatalf("invalid pprof format: %v", err)
				}

				// CPU profile contains execution time samples
				t.Logf("CPU profile: %d samples, %d locations", len(p.Sample), len(p.Location))
				for _, st := range p.SampleType {
					t.Logf("  Sample type: %s/%s", st.Type, st.Unit)
				}
				t.Logf("Duration: %v ns", p.DurationNanos)
				
				// CPU profiles have sample types like:
				// - samples/count (number of samples taken)
				// - cpu/nanoseconds (CPU time in nanoseconds)
				if len(p.SampleType) == 0 {
					t.Error("cpu profile has no sample types")
				}
				
				// Should have captured samples from crypto work
				if len(p.Sample) > 0 {
					t.Logf("Captured %d CPU samples", len(p.Sample))
				}
			})

			server.Stop(nil)

			select {
			case <-server.done:
			case <-time.After(1 * time.Second):
				t.Fatal("server did not stop in time")
			}
		})
	}
}
