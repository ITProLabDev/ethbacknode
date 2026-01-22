package watchdog

import "errors"

// Error definitions for watchdog service operations.
var (
	// ErrConfigStorageEmpty is returned when config storage is not configured.
	ErrConfigStorageEmpty = errors.New("config storage not set")
	// ErrAddressPoolNotSet is returned when the address manager is not configured.
	ErrAddressPoolNotSet = errors.New("address pool not set")
	// ErrChainClientNotSet is returned when the blockchain client is not configured.
	ErrChainClientNotSet = errors.New("blockchain client not set")
)
