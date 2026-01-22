package security

import "errors"

// Error definitions for security operations.
var (
	// ErrConfigStorageEmpty is returned when config storage is not configured.
	ErrConfigStorageEmpty = errors.New("config storage is empty")
)
