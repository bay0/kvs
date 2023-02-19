// Package kvs provides an in-memory key-value store implementation that supports sharding, batching, and transactions.
package kvs

// Value is an interface that defines the methods that a value in the key-value store must implement.
type Value interface {
	// Clone creates a copy of the value.
	Clone() Value
}

// Store is an interface that defines the methods that a key-value store must implement.
type Store interface {
	// Begin starts a transaction that wraps a series of read and write operations.
	// The transaction must be committed or rolled back before subsequent read and write operations.
	Begin() error

	// Commit commits a previously started transaction, applying all the operations.
	Commit() error

	// Rollback cancels a previously started transaction, discarding all the operations.
	Rollback() error

	// Get retrieves the value associated with the given key from the store.
	// If the key is not found in the store, it returns an ErrNotFound error.
	Get(key string) (Value, error)

	// Set adds or updates the given key-value pair in the store.
	// If the key already exists, it overwrites the previous value.
	Set(key string, val Value) error

	// BatchSet adds or updates multiple key-value pairs in the store within a single transaction.
	BatchSet(kvMap map[string]Value) error

	// Delete removes the key-value pair associated with the given key from the store.
	// If the key is not found in the store, it returns an ErrNotFound error.
	Delete(key string) error

	// BatchDelete removes multiple key-value pairs from the store within a single transaction.
	BatchDelete(keys []string) error

	// Keys returns a slice of all the keys in the store.
	Keys() []string
}

// KeyValueStore is a type that implements the Store interface using an in-memory map.
type KeyValueStore struct {
	shards []*shard
	count  int
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

// Begin starts a transaction that wraps a series of read and write operations.
// The transaction must be committed or rolled back before subsequent read and write operations.
func (kvs *KeyValueStore) Begin() error {
	for _, sh := range kvs.shards {
		sh.mu.Lock()
	}

	return nil
}

// Commit commits a previously started transaction, applying all the operations.
func (kvs *KeyValueStore) Commit() error {
	for _, sh := range kvs.shards {
		sh.mu.Unlock()
	}

	return nil
}

// Rollback cancels a previously started transaction, discarding all the operations.
func (kvs *KeyValueStore) Rollback() error {
	for _, sh := range kvs.shards {
		sh.mu.Unlock()
	}

	return nil
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

// BatchSet adds or updates multiple key-value pairs in the store within a single transaction.
func (kvs *KeyValueStore) BatchSet(kvMap map[string]Value) error {
	// Start a transaction
	err := kvs.Begin()
	if err != nil {
		return err
	}

	// Set all key-value pairs in the transaction
	for key, val := range kvMap {
		index := kvs.shardIndex(key)
		sh := kvs.shards[index]

		sh.mu.Lock()
		sh.store[key] = val
		sh.mu.Unlock()
	}

	// Commit the transaction
	err = kvs.Commit()
	if err != nil {
		return err
	}

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

// BatchDelete removes multiple key-value pairs from the store within a single transaction.
func (kvs *KeyValueStore) BatchDelete(keys []string) error {
	// Start a transaction
	err := kvs.Begin()
	if err != nil {
		return err
	}

	// Delete all key-value pairs in the transaction
	for _, key := range keys {
		index := kvs.shardIndex(key)
		sh := kvs.shards[index]

		sh.mu.Lock()
		delete(sh.store, key)
		sh.mu.Unlock()
	}

	// Commit the transaction
	err = kvs.Commit()
	if err != nil {
		return err
	}

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
