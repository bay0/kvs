package kvs

import (
	"sync"
)

// shard represents a partition of the key-value store.
type shard struct {
	id    int
	mu    sync.RWMutex
	store map[string]Value
}

// Keys returns a slice of all the keys in the shard.
func (s *shard) Keys() ([]string, error) {
	keys := make([]string, 0, len(s.store))
	for k := range s.store {
		keys = append(keys, k)
	}

	return keys, nil
}

// Size returns the size of the shard in human-readable format.
func (s *shard) Size() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return formatSize(uint64(len(s.store)))
}
