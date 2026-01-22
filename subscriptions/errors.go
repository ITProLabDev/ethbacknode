package subscriptions

import "errors"

// Error definitions for subscription operations.
var (
	// ErrConfigStorageEmpty is returned when config storage is not configured.
	ErrConfigStorageEmpty = errors.New("config storage not set")
	// ErrUnknownTransaction is returned when a transaction is not found.
	ErrUnknownTransaction = errors.New("unknown transaction")
	// ErrUnknownServiceId is returned when a service ID is not recognized.
	ErrUnknownServiceId = errors.New("unknown serviceId")
)
