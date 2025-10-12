package kvs

import "sync/atomic"

// BatchGet retrieves multiple keys in a single operation with optimized locking.
// Returns successful retrievals in Values map and per-key errors in Errors map.
//
// Thread-safe: Acquires RLock per shard (not per key)
// Complexity: O(N + M) where N=keys, M=shards touched
// TTL-aware: Expired keys return ErrNotFound
// Partial success: One key's error doesn't affect others
func (kvs *KeyValueStore) BatchGet(keys []string) (*BatchGetResult, error) {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return nil, ErrStoreClosed
	}

	result := &BatchGetResult{
		Values: make(map[string]Value),
		Errors: make(map[string]error),
	}

	// Handle empty input
	if len(keys) == 0 {
		return result, nil
	}

	// Group keys by shard to minimize locking
	shardMap := make(map[int][]string)
	for _, key := range keys {
		shardID := kvs.shardIndex(key)
		shardMap[shardID] = append(shardMap[shardID], key)
	}

	// Process each shard with a single lock
	for shardID, shardKeyList := range shardMap {
		sh := kvs.shards[shardID]

		sh.mu.RLock()
		for _, key := range shardKeyList {
			if val, ok := sh.store[key]; ok {
				// Only add to Values if not already present (handles duplicates)
				if _, exists := result.Values[key]; !exists {
					result.Values[key] = val
				}
			} else {
				// Only add to Errors if not in Values (handles duplicates)
				if _, exists := result.Values[key]; !exists {
					result.Errors[key] = ErrNotFound
				}
			}
		}
		sh.mu.RUnlock()
	}

	return result, nil
}

// BatchSet stores multiple key-value pairs with optimized locking.
// Returns count of successful operations and per-key errors.
//
// Thread-safe: Acquires Lock per shard (not per key)
// Complexity: O(N + M) where N=items, M=shards touched
// Partial success: One key's error doesn't affect others
// Duplicate keys: Last value in input map wins
func (kvs *KeyValueStore) BatchSet(items map[string]Value) (*BatchSetResult, error) {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return nil, ErrStoreClosed
	}

	result := &BatchSetResult{
		Successful: 0,
		Errors:     make(map[string]error),
	}

	// Handle empty input
	if len(items) == 0 {
		return result, nil
	}

	// Group items by shard
	shardMap := make(map[int]map[string]Value)
	for key, val := range items {
		shardID := kvs.shardIndex(key)
		if shardMap[shardID] == nil {
			shardMap[shardID] = make(map[string]Value)
		}
		shardMap[shardID][key] = val
	}

	// Process each shard with a single lock
	for shardID, shardItems := range shardMap {
		sh := kvs.shards[shardID]

		sh.mu.Lock()
		for key, val := range shardItems {
			// Check if key is new (for atomic counter)
			_, exists := sh.store[key]
			sh.store[key] = val

			// Remove any TTL when using BatchSet
			delete(sh.ttls, key)

			// Increment counter only for new keys
			if !exists {
				atomic.AddInt64(&kvs.keyCount, 1)
			}

			result.Successful++
		}
		sh.mu.Unlock()
	}

	return result, nil
}

// BatchDelete removes multiple keys with optimized locking.
// Returns count of successful deletions and per-key errors.
//
// Thread-safe: Acquires Lock per shard (not per key)
// Complexity: O(N + M) where N=keys, M=shards touched
// Partial success: One key's error doesn't affect others
// Missing keys: Reported in Errors map with ErrNotFound
func (kvs *KeyValueStore) BatchDelete(keys []string) (*BatchDeleteResult, error) {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return nil, ErrStoreClosed
	}

	result := &BatchDeleteResult{
		Deleted: 0,
		Errors:  make(map[string]error),
	}

	// Handle empty input
	if len(keys) == 0 {
		return result, nil
	}

	// Group keys by shard
	shardMap := make(map[int][]string)
	for _, key := range keys {
		shardID := kvs.shardIndex(key)
		shardMap[shardID] = append(shardMap[shardID], key)
	}

	// Process each shard with a single lock
	for shardID, shardKeyList := range shardMap {
		sh := kvs.shards[shardID]

		sh.mu.Lock()
		for _, key := range shardKeyList {
			if _, ok := sh.store[key]; ok {
				delete(sh.store, key)
				delete(sh.ttls, key) // Remove TTL if present
				atomic.AddInt64(&kvs.keyCount, -1)
				result.Deleted++
			} else {
				// Only report first error for duplicate keys
				if _, alreadyErrored := result.Errors[key]; !alreadyErrored {
					result.Errors[key] = ErrNotFound
				}
			}
		}
		sh.mu.Unlock()
	}

	return result, nil
}
