package kvs

import (
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync/atomic"
	"time"
)

// Snapshot file format structures
const (
	snapshotMagic   = "KVS1"
	snapshotVersion = 1
)

type snapshotHeader struct {
	Magic     string
	Version   uint32
	Timestamp int64
	NumShards uint32
}

type snapshotEntry struct {
	Key        string
	HasTTL     bool
	Expiration time.Time
	Value      Value
}

// SaveSnapshot writes the current store contents to a file with a consistent point-in-time view.
// The snapshot includes all keys, values, and TTL metadata with CRC32 integrity checking.
//
// Thread-safe: Acquires RLock per shard (concurrent with reads)
// Complexity: O(N * S) where N=keys per shard, S=gob encode time
// TTL persistence: Absolute expiration timestamps saved
// File format: Versioned binary with gob encoding
func (kvs *KeyValueStore) SaveSnapshot(filepath string) error {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return ErrStoreClosed
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer file.Close()

	// Create CRC32 hasher
	crc := crc32.NewIEEE()

	// Create encoder that writes to both file and CRC
	encoder := gob.NewEncoder(file)

	// Write header
	header := snapshotHeader{
		Magic:     snapshotMagic,
		Version:   snapshotVersion,
		Timestamp: time.Now().Unix(),
		NumShards: uint32(kvs.count),
	}

	if err := encoder.Encode(header); err != nil {
		return fmt.Errorf("failed to encode header: %w", err)
	}

	// Write each shard's data
	for i, sh := range kvs.shards {
		sh.mu.RLock()

		// Create shard snapshot
		entries := make([]snapshotEntry, 0, len(sh.store))
		for key, val := range sh.store {
			entry := snapshotEntry{
				Key:   key,
				Value: val,
			}

			// Include TTL if present
			if expiration, hasTTL := sh.ttls[key]; hasTTL {
				entry.HasTTL = true
				entry.Expiration = expiration
			}

			entries = append(entries, entry)
		}

		sh.mu.RUnlock()

		// Encode shard data
		shardData := struct {
			ShardID  int
			KeyCount int
			Entries  []snapshotEntry
		}{
			ShardID:  i,
			KeyCount: len(entries),
			Entries:  entries,
		}

		if err := encoder.Encode(shardData); err != nil {
			return fmt.Errorf("failed to encode shard %d: %w", i, err)
		}
	}

	// Calculate CRC32 of the file content
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %w", err)
	}
	if _, err := io.Copy(crc, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Write CRC32 checksum at the end
	checksum := crc.Sum32()
	checksumData := struct {
		Checksum uint32
	}{
		Checksum: checksum,
	}

	// Re-open file in append mode to add checksum
	file2, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen file for checksum: %w", err)
	}
	defer file2.Close()

	encoder2 := gob.NewEncoder(file2)
	if err := encoder2.Encode(checksumData); err != nil {
		return fmt.Errorf("failed to encode checksum: %w", err)
	}

	return nil
}

// LoadSnapshot replaces the current store contents with data from a snapshot file.
// All existing data is cleared before loading (replace semantics).
// Expired keys in the snapshot are skipped (not restored).
//
// Thread-safe: Acquires Lock on all shards (exclusive access)
// Complexity: O(N * D) where N=keys, D=gob decode time
// Atomicity: All-or-nothing (corruption or error leaves store unchanged)
// Version check: Rejects unsupported snapshot formats
func (kvs *KeyValueStore) LoadSnapshot(filepath string) error {
	// Check if store is closed
	if atomic.LoadInt32(&kvs.closed) == 1 {
		return ErrStoreClosed
	}

	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	// Read and validate header
	var header snapshotHeader
	if err := decoder.Decode(&header); err != nil {
		return fmt.Errorf("failed to decode header: %w", err)
	}

	if header.Magic != snapshotMagic {
		return ErrCorruptedSnapshot
	}

	if header.Version != snapshotVersion {
		return ErrUnsupportedVersion
	}

	// Check that numShards matches
	if int(header.NumShards) != kvs.count {
		return fmt.Errorf("snapshot has %d shards, store has %d", header.NumShards, kvs.count)
	}

	// Read all shard data into temporary storage
	type tempShardData struct {
		entries []snapshotEntry
	}

	tempData := make([]tempShardData, kvs.count)
	now := time.Now()

	for i := 0; i < kvs.count; i++ {
		var shardData struct {
			ShardID  int
			KeyCount int
			Entries  []snapshotEntry
		}

		if err := decoder.Decode(&shardData); err != nil {
			return fmt.Errorf("failed to decode shard %d: %w", i, err)
		}

		// Filter out expired keys
		validEntries := make([]snapshotEntry, 0, len(shardData.Entries))
		for _, entry := range shardData.Entries {
			// Skip expired keys
			if entry.HasTTL && now.After(entry.Expiration) {
				continue
			}
			validEntries = append(validEntries, entry)
		}

		tempData[shardData.ShardID] = tempShardData{
			entries: validEntries,
		}
	}

	// Now atomically replace all shard data
	// Lock all shards first
	for _, sh := range kvs.shards {
		sh.mu.Lock()
	}

	// Clear existing data and reset counter
	totalKeys := int64(0)
	for i, sh := range kvs.shards {
		// Clear existing data
		sh.store = make(map[string]Value)
		sh.ttls = make(map[string]time.Time)

		// Load new data
		for _, entry := range tempData[i].entries {
			sh.store[entry.Key] = entry.Value
			if entry.HasTTL {
				sh.ttls[entry.Key] = entry.Expiration
			}
			totalKeys++
		}
	}

	// Update key count
	atomic.StoreInt64(&kvs.keyCount, totalKeys)

	// Unlock all shards
	for _, sh := range kvs.shards {
		sh.mu.Unlock()
	}

	return nil
}
