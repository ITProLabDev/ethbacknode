package txcache

import "errors"

// Error definitions for transaction cache operations.
var (
	// ErrConfigStorageEmpty is returned when config storage is not configured.
	ErrConfigStorageEmpty = errors.New("config storage is empty")
	// ErrUnknownTransaction is returned when a transaction is not found in cache.
	ErrUnknownTransaction = errors.New("unknown transaction")
)
