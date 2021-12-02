package sharder

import (
	"hash"
	"hash/fnv"
	"sync"
)

// Hasher is the common interface to hash a given key.
type Hasher interface {
	Hash(key string) uint32
}

// Opt is used to customer the Sharder's Hasher.
type Opt func(*Sharder)

// WithHasher allows for custom Hasher implementations.
func WithHasher(f Hasher) Opt {
	return func(s *Sharder) {
		s.factory = f
	}
}

// WithLockingHasher is the default Hasher, implemented using a locking fnv32 algorithm.
func WithLockingHasher() Opt {
	return WithHasher(&lockingHasher{
		mu:     sync.Mutex{},
		hasher: fnv.New32a(),
	})
}

// WithLockFreeHasher implements the Hasher interface using lock-free fnv32.
func WithLockFreeHasher() Opt {
	return WithHasher(&lockFreeHasher{})
}

type lockFreeHasher struct {
}

func (f *lockFreeHasher) Hash(key string) uint32 {
	hasher := fnv.New32a()

	if _, err := hasher.Write([]byte(key)); err != nil {
		panic(err)
	}

	return hasher.Sum32()
}

type lockingHasher struct {
	mu     sync.Mutex
	hasher hash.Hash32
}

func (f *lockingHasher) Hash(key string) uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.hasher.Reset()
	if _, err := f.hasher.Write([]byte(key)); err != nil {
		panic(err)
	}

	return f.hasher.Sum32()
}

// New returns a new Sharder with the specified number of shards.
func New(total int, opts ...Opt) *Sharder {
	if total < 1 {
		panic("trying to create Sharder where total < 1")
	}

	if len(opts) == 0 {
		opts = append(opts, WithLockingHasher())
	}

	s := &Sharder{
		total: total,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Sharder determines the shard number for a key.
type Sharder struct {
	total   int
	factory Hasher
}

// Index returns a shard index for the given key. The index is in the range 0..total exclusive.
func (s *Sharder) Index(key string) int {
	i := int(s.factory.Hash(key)) % s.total
	if i < 0 {
		i = -i
	}
	return i
}
