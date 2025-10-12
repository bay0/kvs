package kvs

import "fmt"

// ErrCode is an enumeration of error codes for the key-value store.
type ErrCode int

const (
	ErrUnknown ErrCode = iota
	ErrNotFound
	ErrDuplicate
	ErrInvalidNumShards
	ErrInvalidShardIndex
	ErrCorruptedSnapshot
	ErrUnsupportedVersion
	ErrInvalidTTL
	ErrStoreClosed
	ErrNilHashFunc
)

var errMsg = map[ErrCode]string{
	ErrUnknown:            "unknown error",
	ErrNotFound:           "item not found",
	ErrDuplicate:          "item already exists",
	ErrInvalidNumShards:   "invalid number of shards",
	ErrInvalidShardIndex:  "hash function returned invalid shard index",
	ErrCorruptedSnapshot:  "snapshot file is corrupted",
	ErrUnsupportedVersion: "snapshot version not supported",
	ErrInvalidTTL:         "TTL duration must be non-negative",
	ErrStoreClosed:        "store has been closed",
	ErrNilHashFunc:        "hash function cannot be nil",
}

// Error returns the string representation of an error code.
func (c ErrCode) Error() string {
	return fmt.Sprintf("kvs: %v", errMsg[c])
}
