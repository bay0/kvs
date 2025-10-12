# kvs

A high-performance, thread-safe key-value store library for Go with sharding, TTL support, batch operations, and persistence.

## Features

- **Thread-Safe Operations**: Full concurrency support with fine-grained shard-level locking
- **Sharding**: Configurable sharding for improved concurrent performance
- **TTL Support**: Redis-like key expiration with hybrid active/passive cleanup
- **Batch Operations**: Optimized batch get/set/delete operations with per-shard locking
- **Custom Hash Functions**: Pluggable hash functions for flexible key distribution
- **Snapshot Persistence**: Save and load store state to/from disk with integrity checking
- **Zero External Dependencies**: Built entirely on Go standard library
- **Race-Tested**: All operations validated with Go's race detector

## Installation

```bash
go get github.com/bay0/kvs
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/bay0/kvs"
)

type Person struct {
    Name string
    Age  int
}

func (p Person) Clone() kvs.Value {
    return Person{
        Name: p.Name,
        Age:  p.Age,
    }
}

func main() {
    // Create a new key-value store with 10 shards
    store, err := kvs.NewKeyValueStore(10)
    if err != nil {
        panic(err)
    }
    defer store.Close() // Important: Close to stop background goroutines

    // Basic operations
    person := Person{Name: "Alice", Age: 30}

    // Set a value
    store.Set("alice", person)

    // Check if key exists (no error handling needed)
    if store.Exists("alice") {
        fmt.Println("Alice exists in store")
    }

    // Get the total number of keys (O(1) operation)
    fmt.Printf("Store contains %d keys\n", store.Len())

    // Get a value
    val, err := store.Get("alice")
    if err != nil {
        panic(err)
    }
    p := val.(Person)
    fmt.Printf("%s is %d years old\n", p.Name, p.Age)

    // Delete a key
    store.Delete("alice")
}
```

## Core Operations

### Basic Operations

```go
// Set a key-value pair
err := store.Set("key", value)

// Get a value
val, err := store.Get("key")

// Delete a key
err := store.Delete("key")

// Check if a key exists (returns bool, never errors)
exists := store.Exists("key")

// Get the total number of keys (O(1) operation)
count := store.Len()

// Get all keys in the store
keys, err := store.Keys()
```

### TTL Support

Keys can be set with automatic expiration. The store provides both active cleanup (background goroutine every 50ms) and passive cleanup (on access).

```go
// Set a key with 5-minute TTL
err := store.SetWithTTL("session", sessionData, 5*time.Minute)

// Set a key with zero TTL (no expiration)
err := store.SetWithTTL("permanent", data, 0)

// Negative TTL returns ErrInvalidTTL
err := store.SetWithTTL("bad", data, -1*time.Second) // Error!

// Close the store to stop background cleanup goroutine
store.Close()
```

**TTL Behavior:**
- TTL precision: ±50ms (background cleanup runs every 50ms)
- Zero TTL: Treated as no expiration
- Negative TTL: Returns `ErrInvalidTTL`
- Expired keys: Return `ErrNotFound` on access (passive cleanup)
- Background cleanup: Active removal every 50ms

### Batch Operations

Batch operations optimize performance by acquiring locks once per shard rather than once per key.

```go
// Batch Get - retrieve multiple keys at once
result, err := store.BatchGet([]string{"key1", "key2", "key3"})
if err != nil {
    panic(err)
}

// Check results
for key, value := range result.Values {
    fmt.Printf("%s: %v\n", key, value)
}

// Check errors for individual keys
for key, err := range result.Errors {
    fmt.Printf("%s failed: %v\n", key, err)
}

// Batch Set - store multiple key-value pairs at once
items := map[string]kvs.Value{
    "key1": value1,
    "key2": value2,
    "key3": value3,
}
result, err := store.BatchSet(items)
fmt.Printf("Successfully set %d keys\n", result.Successful)

// Batch Delete - remove multiple keys at once
result, err := store.BatchDelete([]string{"key1", "key2", "key3"})
fmt.Printf("Successfully deleted %d keys\n", result.Deleted)
```

**Batch Operation Features:**
- **Partial Success**: One key's error doesn't affect others
- **Per-Shard Locking**: Locks acquired once per shard, not per key
- **Complexity**: O(N + M) where N=items, M=shards touched
- **Thread-Safe**: Concurrent batch operations are safe

### Custom Hash Functions

You can provide a custom hash function for key distribution across shards.

```go
// Custom hash function using modulo
customHash := func(key string) int {
    sum := 0
    for _, ch := range key {
        sum += int(ch)
    }
    return sum % 10
}

// Create store with custom hash
store, err := kvs.NewKeyValueStoreWithHash(10, customHash)
if err != nil {
    panic(err)
}
defer store.Close()

// Your hash function MUST:
// - Be deterministic (same key always returns same index)
// - Return values in range [0, numShards)
// - Be thread-safe (may be called concurrently)
```

**Default Hash Function:** FNV-1a hashing

### Snapshot Persistence

Save and load the entire store state to/from disk with integrity checking.

```go
// Save snapshot to disk
err := store.SaveSnapshot("/path/to/snapshot.kvs")
if err != nil {
    panic(err)
}

// Load snapshot from disk
newStore, _ := kvs.NewKeyValueStore(10)
err = newStore.LoadSnapshot("/path/to/snapshot.kvs")
if err != nil {
    panic(err)
}
defer newStore.Close()
```

**Snapshot Features:**
- **Point-in-Time Consistency**: Snapshot captures consistent state
- **TTL Preservation**: Absolute expiration timestamps are saved
- **Replace Semantics**: LoadSnapshot clears existing data
- **Expired Key Filtering**: Expired keys are skipped during load
- **Integrity Checking**: CRC32 checksums detect corruption
- **Versioned Format**: Magic bytes and version validation
- **Thread-Safe**: Concurrent with reads, uses per-shard RLock

**File Format:**
- Magic bytes: "KVS1"
- Version: 1
- Encoding: Gob (Go's binary encoding)
- Checksum: CRC32 (IEEE)

## Error Codes

The library defines the following error codes:

```go
const (
    ErrUnknown              // Unknown error
    ErrNotFound             // Key not found in store
    ErrDuplicate            // Key already exists (currently unused)
    ErrInvalidNumShards     // Invalid number of shards (≤ 0)
    ErrInvalidShardIndex    // Shard index out of bounds
    ErrCorruptedSnapshot    // Snapshot file is corrupted
    ErrUnsupportedVersion   // Snapshot version not supported
    ErrInvalidTTL           // TTL value is invalid (negative)
    ErrStoreClosed          // Operation on closed store
    ErrNilHashFunc          // Nil hash function provided
)
```

## Interfaces

### Value Interface

All values stored in the kvs must implement the `Value` interface:

```go
type Value interface {
    // Clone creates a copy of the value
    Clone() Value
}
```

### Store Interface

The core store interface defines essential operations:

```go
type Store interface {
    Get(key string) (Value, error)
    Set(key string, val Value) error
    Delete(key string) error
    Keys() []string
}
```

## Performance Characteristics

| Operation | Time Complexity | Notes |
|-----------|----------------|-------|
| Set | O(1) | Hash lookup + map insert |
| Get | O(1) | Hash lookup + map access |
| Delete | O(1) | Hash lookup + map delete |
| Exists | O(1) | Hash lookup + map check |
| Len | O(1) | Atomic counter read |
| Keys | O(N) | N = total keys |
| BatchGet | O(N + M) | N = keys, M = shards touched |
| BatchSet | O(N + M) | N = items, M = shards touched |
| BatchDelete | O(N + M) | N = keys, M = shards touched |
| SaveSnapshot | O(N × S) | N = keys, S = gob encode time |
| LoadSnapshot | O(N × D) | N = keys, D = gob decode time |

## Thread Safety

All operations are thread-safe:

- **Read Operations** (Get, Exists, BatchGet): Use RLock (concurrent with other reads)
- **Write Operations** (Set, Delete, BatchSet, BatchDelete): Use Lock (exclusive access)
- **Shard-Level Locking**: Each shard has its own lock for improved concurrency
- **Atomic Operations**: Len() uses atomic operations for lock-free reads
- **Race Tested**: All operations validated with `-race` flag

## Best Practices

1. **Always Close the Store**: Call `store.Close()` to stop background TTL cleanup goroutines and prevent leaks
   ```go
   store, _ := kvs.NewKeyValueStore(10)
   defer store.Close()
   ```

2. **Choose Appropriate Shard Count**: More shards = better concurrency, but diminishing returns after ~10-100 shards depending on workload

3. **Use Batch Operations**: When working with multiple keys, batch operations are significantly faster
   ```go
   // Instead of multiple Get calls:
   result, _ := store.BatchGet([]string{"key1", "key2", "key3"})
   ```

4. **Check for Store Closed**: After closing, all operations return `ErrStoreClosed`
   ```go
   if err == kvs.ErrStoreClosed {
       // Handle closed store
   }
   ```

5. **Handle TTL Cleanup**: Background cleanup runs every 50ms, providing ±50ms precision

6. **Implement Clone Correctly**: Ensure your Value.Clone() creates a true deep copy
   ```go
   func (p Person) Clone() kvs.Value {
       return Person{Name: p.Name, Age: p.Age} // Deep copy
   }
   ```

## Examples

See the [examples/](examples/) directory for complete working examples:

- **Basic Usage**: Simple get/set/delete operations
- **Multi-threaded**: Concurrent access patterns
- **Sharding with Consistent Hashing**: Custom hash function example
- **TTL Management**: Working with key expiration
- **Batch Operations**: Optimized bulk operations
- **Snapshot Persistence**: Save and restore store state

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please ensure:
- All tests pass: `go test -v -race ./...`
- Code is formatted: `go fmt ./...`
- Linter passes: `golangci-lint run`
- New features include tests and documentation
