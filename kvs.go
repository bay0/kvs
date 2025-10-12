// Package kvs provides an in-memory key-value store implementation that supports sharding, batching, and transactions.
package kvs

import (
	"context"
	"sync/atomic"
	"time"
)

// Value is an interface that defines the methods that a value in the key-value store must implement.
type Value interface {
	// Clone creates a copy of the value.
	Clone() Value
}

// Store is an interface that defines the methods that a key-value store must implement.
type Store interface {
	// Get retrieves the value associated with the given key from the store.
	// If the key is not found in the store, it returns an ErrNotFound error.
	Get(key string) (Value, error)

	// Set adds or updates the given key-value pair in the store.
	// If the key already exists, it overwrites the previous value.
	Set(key string, val Value) error

	// Delete removes the key-value pair associated with the given key from the store.
	// If the key is not found in the store, it returns an ErrNotFound error.
	Delete(key string) error

	// Keys returns a slice of all the keys in the store.
	Keys() []string
}

// HashFunc maps a key string to a shard index.
// Must be deterministic and return values in range [0, numShards).
type HashFunc func(key string) int

// BatchGetResult contains the results of a BatchGet operation.
// Keys appear in either Values (success) or Errors (failure), never both.
type BatchGetResult struct {
	Values map[string]Value // Successful retrievals: key → value
	Errors map[string]error // Failed retrievals: key → error
}

// BatchSetResult contains the results of a BatchSet operation.
// Successful count + len(Errors) == total input items.
type BatchSetResult struct {
	Successful int              // Count of successfully stored keys
	Errors     map[string]error // Failed operations: key → error
}

// BatchDeleteResult contains the results of a BatchDelete operation.
// Deleted count + len(Errors) == total input keys.
type BatchDeleteResult struct {
	Deleted int              // Count of successfully deleted keys
	Errors  map[string]error // Failed deletions: key → error
}

// KeyValueStore is a type that implements the Store interface using an in-memory map.
type KeyValueStore struct {
	shards       []*shard
	count        int
	hashFunc     HashFunc // Custom hash function for key distribution
	keyCount     int64    // Atomic counter for total key count (for O(1) Len operation)
	cleanupCtx   context.Context
	cleanupStop  context.CancelFunc
	closed       int32 // Atomic flag: 1 if closed, 0 if open
	cleanupDone  chan struct{}
}

// defaultHashFunc implements FNV-1a hashing for key distribution.
func defaultHashFunc(numShards int) HashFunc {
	return func(key string) int {
		var h uint32 = 2166136261 // FNV-1a offset basis
		for i := 0; i < len(key); i++ {
			h = (h * 16777619) ^ uint32(key[i])
		}
		return int(h) % numShards
	}
}

// NewKeyValueStore creates a new KeyValueStore instance with a specified number of shards.
// Uses the default FNV-1a hash function for key distribution.
// Starts a background goroutine for TTL cleanup - call Close() to stop it.
func NewKeyValueStore(numShards int) (*KeyValueStore, error) {
	return NewKeyValueStoreWithHash(numShards, defaultHashFunc(numShards))
}

// NewKeyValueStoreWithHash creates a new store with a custom hash function.
// The hash function maps keys to shard indices and must be deterministic.
//
// Parameters:
//   - numShards: Number of shards (must be > 0)
//   - hashFunc: Custom hash function (must return [0, numShards))
//
// Returns error if numShards <= 0 or hashFunc is nil.
// Default behavior: Use NewKeyValueStore() for FNV-1a hash.
//
// IMPORTANT: The hash function MUST:
//   - Be deterministic (same key always returns same index)
//   - Return values in range [0, numShards)
//   - Be thread-safe (may be called concurrently)
func NewKeyValueStoreWithHash(numShards int, hashFunc HashFunc) (*KeyValueStore, error) {
	if numShards <= 0 {
		return nil, ErrInvalidNumShards
	}

	if hashFunc == nil {
		return nil, ErrNilHashFunc
	}

	shards := make([]*shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &shard{
			id:    i,
			store: make(map[string]Value),
			ttls:  make(map[string]time.Time),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	kvs := &KeyValueStore{
		shards:      shards,
		count:       numShards,
		hashFunc:    hashFunc,
		keyCount:    0,
		cleanupCtx:  ctx,
		cleanupStop: cancel,
		closed:      0,
		cleanupDone: make(chan struct{}),
	}

	// Start TTL cleanup goroutine
	go kvs.ttlCleanup()

	return kvs, nil
}

// shardIndex returns the index of the shard that should contain a given key.
func (kvs *KeyValueStore) shardIndex(key string) int {
	return kvs.hashFunc(key)
}

// Set adds or updates the given key-value pair in the store.
// If the key already exists, it overwrites the previous value.
func (kvs *KeyValueStore) Set(key string, val Value) error {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return ErrStoreClosed
	}

	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Check if key is new (for atomic counter)
	_, exists := sh.store[key]
	sh.store[key] = val

	// Remove any existing TTL when using regular Set
	delete(sh.ttls, key)

	// Increment counter only for new keys
	if !exists {
		atomic.AddInt64(&kvs.keyCount, 1)
	}

	return nil
}

// Get retrieves the value associated with the given key from the store.
// If the key is not found in the store, it returns an error.
// TTL-aware: Returns ErrNotFound for expired keys (passive cleanup).
func (kvs *KeyValueStore) Get(key string) (Value, error) {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return nil, ErrStoreClosed
	}

	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.RLock()
	val, ok := sh.store[key]
	expired := false
	if ok {
		expired = kvs.isExpired(sh, key)
	}
	sh.mu.RUnlock()

	if !ok || expired {
		// If expired, perform passive cleanup
		if expired {
			sh.mu.Lock()
			delete(sh.store, key)
			delete(sh.ttls, key)
			atomic.AddInt64(&kvs.keyCount, -1)
			sh.mu.Unlock()
		}
		return nil, ErrNotFound
	}

	return val, nil
}

// Delete removes the key-value pair associated with the given key from the store.
// If the key is not found in the store, it returns an error.
func (kvs *KeyValueStore) Delete(key string) error {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return ErrStoreClosed
	}

	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if _, ok := sh.store[key]; !ok {
		return ErrNotFound
	}

	delete(sh.store, key)
	delete(sh.ttls, key) // Remove TTL if present

	// Decrement counter after successful deletion
	atomic.AddInt64(&kvs.keyCount, -1)

	return nil
}

// Keys returns a slice of all the keys in the store.
func (kvs *KeyValueStore) Keys() ([]string, error) {
	keys := make([]string, 0)

	for _, sh := range kvs.shards {
		sh.mu.RLock()
		shKeys, err := sh.Keys()
		sh.mu.RUnlock()
		if err != nil {
			return nil, err
		}
		keys = append(keys, shKeys...)
	}

	return keys, nil
}

// Size returns the size of the store in human-readable format.
func (kvs *KeyValueStore) Size() string {
	var totalSize uint64

	for _, sh := range kvs.shards {
		sh.mu.RLock()
		size := uint64(len(sh.store))
		totalSize += size
		sh.mu.RUnlock()
	}

	return formatSize(totalSize)
}
