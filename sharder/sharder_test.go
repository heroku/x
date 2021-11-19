package sharder_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/heroku/x/sharder"
)

type TestHasher struct {
	expected uint32
}

func (f *TestHasher) Hash(key string) uint32 {
	return f.expected
}

//nolint:gocyclo // This is a test, there are going to be lots of if statements
func TestSharder(t *testing.T) {
	t.Run("constructing with bad count panics", func(t *testing.T) {
		defer func() {
			if x := recover(); x == nil {
				t.Fatal("wanted New to panic when given bad count")
			}
		}()

		sharder.New(0)
	})

	t.Run("custom hasher", func(t *testing.T) {
		// fix the test so that all inputs return 7
		f := &TestHasher{expected: 1337}
		s := sharder.New(10, sharder.WithHasher(f))

		for i := 0; i < 100; i++ {
			if idx := s.Index(uuid.New().String()); idx != 7 {
				t.Fatalf("want index 0, got %d", idx)
			}
		}
	})

	t.Run("only one shard, default hasher", func(t *testing.T) {
		s := sharder.New(1)
		for i := 0; i < 100; i++ {
			if idx := s.Index(uuid.New().String()); idx != 0 {
				t.Fatalf("want index 0, got %d", idx)
			}
		}
	})

	t.Run("only one shard, locking hasher", func(t *testing.T) {
		s := sharder.New(1, sharder.WithLockingHasher())
		for i := 0; i < 100; i++ {
			if idx := s.Index(uuid.New().String()); idx != 0 {
				t.Fatalf("want index 0, got %d", idx)
			}
		}
	})

	t.Run("only one shard, lock free hasher", func(t *testing.T) {
		s := sharder.New(1, sharder.WithLockFreeHasher())
		for i := 0; i < 100; i++ {
			if idx := s.Index(uuid.New().String()); idx != 0 {
				t.Fatalf("want index 0, got %d", idx)
			}
		}
	})

	t.Run("many shards", func(t *testing.T) {

		locking := sharder.New(10, sharder.WithLockingHasher())
		lockFree := sharder.New(10, sharder.WithLockFreeHasher())
		for i := 0; i < 100; i++ {
			uuid := uuid.New().String()
			lockingIdx := locking.Index(uuid)
			freeIdx := lockFree.Index(uuid)

			if lockingIdx < 0 || lockingIdx >= 10 {
				t.Fatalf("want locking index in range 0..10, got %d", lockingIdx)
			}

			if freeIdx < 0 || freeIdx >= 10 {
				t.Fatalf("want lock free index in range 0..10, got %d", freeIdx)
			}

			if lockingIdx != freeIdx {
				t.Fatalf("want locking index equal to lock free index, got %d == %d", lockingIdx, freeIdx)
			}
		}
	})
}
