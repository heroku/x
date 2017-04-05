package goruntimemetrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	metricWaitTime = 20 * time.Second
)

// Payload is the type that app-ingress is expecting.
type Payload struct {
	Counters map[string]float64 `json:"counters"`
	Gauges   map[string]float64 `json:"gauges"`
}

func init() {
	endpoint := os.Getenv("HEROKU_METRICS_URL")
	if endpoint == "" {
		log.Println("the heroku go metrics subsystem cannot be initialized because there is no target (HEROKU_METRICS_URL unset)")
		return
	}

	go func() {
		t := time.NewTicker(metricWaitTime)
		defer t.Stop()

		for {
			<-t.C

			err := submitPayload(gatherMetrics(), endpoint)
			if err != nil {
				log.Printf("[heroku metrics] error when submitting metrics: %v", err)
				continue
			}
		}
	}()
}

var lastGCPause uint64

func gatherMetrics() *Payload {
	result := &Payload{
		Counters: map[string]float64{},
		Gauges:   map[string]float64{},
	}

	stats := &runtime.MemStats{}
	runtime.ReadMemStats(stats)

	// cribbed from https://github.com/codahale/metrics/blob/master/runtime/memstats.go

	pauseNS := stats.PauseTotalNs - lastGCPause
	lastGCPause = stats.PauseTotalNs

	result.Counters["go.gc.collections"] = float64(stats.NumGC)
	result.Counters["go.gc.pause.ns"] = float64(pauseNS)

	result.Gauges["go.memory.heap.bytes"] = float64(stats.Alloc)
	result.Gauges["go.memory.stack.bytes"] = float64(stats.StackInuse)

	return result
}

func submitPayload(p *Payload, where string) error {
	b := bytes.NewBuffer(make([]byte, 512))
	err := json.NewEncoder(b).Encode(p)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", where, b)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected %v (http.StatusOK) but got %s", http.StatusOK, resp.Status)
	}

	return nil
}
