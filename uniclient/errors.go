package uniclient

import "errors"

// Error definitions for uniclient operations.
var (
	// ErrInvalidBalanceResponse is returned when balance response format is invalid.
	ErrInvalidBalanceResponse = errors.New("invalid balance response")
)
