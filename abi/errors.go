package abi

import "errors"

// Error definitions for ABI operations.
var (
	// ErrSmartContractMethodParamsCountMismatch is returned when ABI params don't match.
	ErrSmartContractMethodParamsCountMismatch = errors.New("method params count mismatch")
	// ErrSmartContractUnknownMethod is returned when a method is not found in ABI.
	ErrSmartContractUnknownMethod = errors.New("unknown method")
	// ErrInvalidParamsData is returned when parameter data cannot be parsed.
	ErrInvalidParamsData = errors.New("invalid params Data")
	// ErrConfigStorageEmpty is returned when config storage is not configured.
	ErrConfigStorageEmpty = errors.New("config storage is empty")
	// ErrUnknownContract is returned when a contract is not in the registry.
	ErrUnknownContract = errors.New("unknown contract")
	// ErrNotTransferMethod is returned when call data is not a transfer method.
	ErrNotTransferMethod = errors.New("not transfer method")
)
