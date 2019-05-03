package testenv

import (
	"testing"
)

// NewKafkaURLs returns the broker addresses for Sarama to connect to a local
// Kafka host at KAFKA_HOST or localhost:9092.
func NewKafkaURLs(t testing.TB) []string {
	t.Helper()

	host, err := getenv("KAFKA_HOST", "localhost:9092")
	if err != nil {
		t.Skip(err.Error())
	}

	return []string{host}
}
