// Package kvs provides a key-value store.
package kvs

import (
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

func (c ErrCode) Error() string {
	return errMsg[c]
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
	mu    sync.RWMutex
	store map[string]Value
}

// NewKeyValueStore creates a new KeyValueStore instance.
func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		store: make(map[string]Value),
	}
}

// Set adds or updates the given key-value pair in the store.
// If the key already exists, it overwrites the previous value.
func (kvs *KeyValueStore) Set(key string, val Value) error {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	kvs.store[key] = val.Clone()
	return nil
}

// Get retrieves the value associated with the given key from the store.
// If the key is not found in the store, it returns an error.
func (kvs *KeyValueStore) Get(key string) (Value, error) {
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()
	val, ok := kvs.store[key]
	if !ok {
		return nil, ErrNotFound
	}
	return val.Clone(), nil
}

// Delete removes the key-value pair associated with the given key from the store.
// If the key is not found in the store, it returns an error.
func (kvs *KeyValueStore) Delete(key string) error {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	if _, ok := kvs.store[key]; !ok {
		return ErrNotFound
	}
	delete(kvs.store, key)
	return nil
}

// Keys returns a slice of all the keys in the store.
func (kvs *KeyValueStore) Keys() []string {
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()
	keys := make([]string, 0, len(kvs.store))
	for k := range kvs.store {
		keys = append(keys, k)
	}
	return keys
}
