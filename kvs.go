// Package kvs provides a key-value store.
package kvs

import (
	"fmt"
	"sync"
)

// ErrCode is an enumeration of error codes for the key-value store.
type ErrCode int

const (
	ErrUnknown ErrCode = iota
	ErrNotFound
	ErrDuplicate
)

var errMsg = map[ErrCode]string{
	ErrUnknown:   "unknown error",
	ErrNotFound:  "item not found",
	ErrDuplicate: "item already exists",
}

// Error returns the string representation of an error code.
func (c ErrCode) Error() string {
	return fmt.Sprintf("kvs: %v", errMsg[c])
}

// Value is an interface that defines the methods that a value in the key-value store must implement.
type Value interface {
	// Clone creates a copy of the value.
	Clone() Value
}

// Store is an interface that defines the methods that a key-value store must implement.
type Store interface {
	// Get retrieves the value associated with the given key from the store.
	// If the key is not found in the store, it returns an error.
	Get(key string) (Value, error)

	// Set adds or updates the given key-value pair in the store.
	// If the key already exists, it overwrites the previous value.
	Set(key string, val Value) error

	// Delete removes the key-value pair associated with the given key from the store.
	// If the key is not found in the store, it returns an error.
	Delete(key string) error

	// Keys returns a slice of all the keys in the store.
	Keys() []string
}

// KeyValueStore is a type that implements the Store interface using an in-memory map.
type KeyValueStore struct {
	shards []*shard
	count  int
}

// shard represents a partition of the key-value store.
type shard struct {
	id    int
	mu    sync.RWMutex
	store map[string]Value
}

// NewKeyValueStore creates a new KeyValueStore instance with a specified number of shards.
func NewKeyValueStore(numShards int) *KeyValueStore {
	shards := make([]*shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &shard{
			id:    i,
			store: make(map[string]Value),
		}
	}

	return &KeyValueStore{
		shards: shards,
		count:  numShards,
	}
}

// shardIndex returns the index of the shard that should contain a given key.
func (kvs *KeyValueStore) shardIndex(key string) int {
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h = (h * 16777619) ^ uint32(key[i])
	}

	return int(h) % kvs.count
}

// Set adds or updates the given key-value pair in the store.
// If the key already exists, it overwrites the previous value.
func (kvs *KeyValueStore) Set(key string, val Value) error {
	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.store[key] = val
	return nil
}

// Get retrieves the value associated with the given key from the store.
// If the key is not found in the store, it returns an error.
func (kvs *KeyValueStore) Get(key string) (Value, error) {
	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.RLock()
	defer sh.mu.RUnlock()

	val, ok := sh.store[key]

	if !ok {
		return nil, ErrNotFound
	}

	return val, nil
}

// Delete removes the key-value pair associated with the given key from the store.
// If the key is not found in the store, it returns an error.
func (kvs *KeyValueStore) Delete(key string) error {
	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if _, ok := sh.store[key]; !ok {
		return ErrNotFound
	}

	delete(sh.store, key)

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

// Keys returns a slice of all the keys in the shard.
func (s *shard) Keys() ([]string, error) {
	keys := make([]string, 0, len(s.store))
	for k := range s.store {
		keys = append(keys, k)
	}

	return keys, nil
}
