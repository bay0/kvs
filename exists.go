package kvs

import "sync/atomic"

// Exists returns true if the key is present in the store, false otherwise.
// This method does NOT return an error for missing keys (unlike Get).
//
// Thread-safe: Uses shard-level RLock
// Complexity: O(1) hash lookup + map access
// TTL-aware: Returns false for expired keys
func (kvs *KeyValueStore) Exists(key string) bool {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return false
	}

	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.RLock()
	_, exists := sh.store[key]
	expired := false
	if exists {
		expired = kvs.isExpired(sh, key)
	}
	sh.mu.RUnlock()

	// Perform passive cleanup if expired
	if expired {
		sh.mu.Lock()
		delete(sh.store, key)
		delete(sh.ttls, key)
		atomic.AddInt64(&kvs.keyCount, -1)
		sh.mu.Unlock()
		return false
	}

	return exists
}

// Len returns the total number of key-value pairs across all shards.
// This operation is O(1) using an atomic counter maintained by Set/Delete.
//
// Thread-safe: Uses atomic load
// Complexity: O(1)
// TTL-aware: Includes keys that haven't been cleaned up yet (eventual consistency)
func (kvs *KeyValueStore) Len() int {
	return int(atomic.LoadInt64(&kvs.keyCount))
}
