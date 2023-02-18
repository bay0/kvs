package kvs

import "fmt"

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
