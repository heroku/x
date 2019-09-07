package sharder

import (
	"hash"
	"hash/fnv"
	"sync"
)

// New returns a new Sharder with the specified number of shards.
func New(total int) *Sharder {
	if total < 1 {
		panic("trying to create Sharder where total < 1")
	}
	return &Sharder{
		total:  total,
		hasher: fnv.New32a(),
	}
}

// Sharder determines the shard number for a key.
type Sharder struct {
	total  int
	mu     sync.Mutex
	hasher hash.Hash32
}

// Index returns a shard index for the given key. The index is in the range 0..total exclusive.
func (s *Sharder) Index(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hasher.Reset()
	if _, err := s.hasher.Write([]byte(key)); err != nil {
		panic(err)
	}

	i := int(s.hasher.Sum32()) % s.total
	if i < 0 {
		i = -i
	}
	return i
}
