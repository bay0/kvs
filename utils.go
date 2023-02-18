package kvs

import "fmt"

// formatSize returns a human-readable string representation of a size in bytes.
func formatSize(size uint64) string {
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

	var suffix string
	var divisor uint64 = 1
	for i := 0; i < len(suffixes); i++ {
		if size < divisor*1024 {
			suffix = suffixes[i]
			break
		}
		divisor *= 1024
	}

	return fmt.Sprintf("%d %s", size/divisor, suffix)
}
