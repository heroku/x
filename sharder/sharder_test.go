package sharder

import (
	"testing"

	"github.com/google/uuid"
)

func TestSharder(t *testing.T) {
	t.Run("constructing with bad count panics", func(t *testing.T) {
		defer func() {
			if x := recover(); x == nil {
				t.Fatal("wanted New to panic when given bad count")
			}
		}()

		New(0)
	})

	t.Run("only one shard", func(t *testing.T) {
		s := New(1)
		for i := 0; i < 100; i++ {
			if idx := s.Index(uuid.New().String()); idx != 0 {
				t.Fatalf("want index 0, got %d", idx)
			}
		}
	})

	t.Run("many shards", func(t *testing.T) {
		s := New(10)
		for i := 0; i < 100; i++ {
			if idx := s.Index(uuid.New().String()); idx < 0 || idx >= 10 {
				t.Fatalf("want index in range 0..10, got %d", idx)
			}
		}
	})
}
