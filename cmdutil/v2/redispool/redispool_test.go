package redispool

import (
	"os"
	"testing"
)

func TestPools(t *testing.T) {
	t.Run("missing attachment url", func(t *testing.T) {
		cfg := Config{
			AttachmentNames: []string{"REDIS_UNKNOWN"},
		}

		if _, err := cfg.Pools(); err == nil {
			t.Fatal("want missing attachment url error, got nil")
		}
	})

	t.Run("two attachments", func(t *testing.T) {
		os.Setenv("REDIS1_URL", "redis://localhost")
		os.Setenv("REDIS2_URL", "redis://localhost")
		defer func() {
			os.Setenv("REDIS1_URL", "")
			os.Setenv("REDIS2_URL", "")
		}()

		cfg := Config{
			AttachmentNames: []string{"REDIS1", "REDIS2"},
		}

		pools, err := cfg.Pools()
		if err != nil {
			t.Fatalf("wanted nil error, got %v", err)
		}

		if want, got := 2, len(pools); want != got {
			t.Fatalf("want %d pools, got %d", want, got)
		}
	})
}
