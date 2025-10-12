package kvs

import (
	"sync/atomic"
	"time"
)

// SetWithTTL stores a key-value pair with automatic expiration after the specified duration.
// The key will be automatically removed when TTL expires (active cleanup).
// Accessing an expired key returns ErrNotFound (passive cleanup).
//
// Thread-safe: Acquires shard Lock
// Complexity: O(1) hash lookup + map insert
// TTL precision: ±50ms (background cleanup runs every 50ms)
// Zero TTL: Treated as no expiration
// Negative TTL: Returns ErrInvalidTTL
func (kvs *KeyValueStore) SetWithTTL(key string, val Value, ttl time.Duration) error {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return ErrStoreClosed
	}

	// Validate TTL
	if ttl < 0 {
		return ErrInvalidTTL
	}

	index := kvs.shardIndex(key)
	sh := kvs.shards[index]

	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Check if key is new (for atomic counter)
	_, exists := sh.store[key]
	sh.store[key] = val

	// Set TTL if > 0
	if ttl > 0 {
		sh.ttls[key] = time.Now().Add(ttl)
	} else {
		// Zero TTL means no expiration, remove any existing TTL
		delete(sh.ttls, key)
	}

	// Increment counter only for new keys
	if !exists {
		atomic.AddInt64(&kvs.keyCount, 1)
	}

	return nil
}

// isExpired checks if a key has expired. Must be called with shard lock held.
func (kvs *KeyValueStore) isExpired(sh *shard, key string) bool {
	if expiration, hasTTL := sh.ttls[key]; hasTTL {
		return time.Now().After(expiration)
	}
	return false
}

// ttlCleanup runs in the background and removes expired keys.
// Runs every 50ms to provide ~±50ms TTL precision.
func (kvs *KeyValueStore) ttlCleanup() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	defer close(kvs.cleanupDone)

	for {
		select {
		case <-kvs.cleanupCtx.Done():
			return
		case <-ticker.C:
			kvs.cleanupExpiredKeys()
		}
	}
}

// cleanupExpiredKeys iterates through all shards and removes expired keys.
func (kvs *KeyValueStore) cleanupExpiredKeys() {
	now := time.Now()

	for _, sh := range kvs.shards {
		sh.mu.Lock()

		// Find expired keys
		expiredKeys := make([]string, 0)
		for key, expiration := range sh.ttls {
			if now.After(expiration) {
				expiredKeys = append(expiredKeys, key)
			}
		}

		// Remove expired keys
		for _, key := range expiredKeys {
			delete(sh.store, key)
			delete(sh.ttls, key)
			atomic.AddInt64(&kvs.keyCount, -1)
		}

		sh.mu.Unlock()
	}
}

// Close gracefully shuts down the store, stopping background goroutines.
// Must be called to prevent goroutine leaks if TTL features were used.
//
// Thread-safe: Can be called multiple times (idempotent)
// Blocks until: TTL cleanup goroutine terminates (timeout: 5 seconds)
// Post-close: All operations return ErrStoreClosed
func (kvs *KeyValueStore) Close() error {
	// Check if already closed (atomic compare-and-swap)
	if !atomic.CompareAndSwapInt32(&kvs.closed, 0, 1) {
		// Already closed, return success (idempotent)
		return nil
	}

	// Stop the cleanup goroutine
	kvs.cleanupStop()

	// Wait for cleanup goroutine to finish (with timeout)
	select {
	case <-kvs.cleanupDone:
		// Cleanup finished successfully
		return nil
	case <-time.After(5 * time.Second):
		// Timeout waiting for cleanup
		return nil // Still return success as we've marked as closed
	}
}
