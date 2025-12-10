package abi

import "errors"

var (
	ErrSmartContractMethodParamsCountMismatch = errors.New("method params count mismatch")
	ErrSmartContractUnknownMethod             = errors.New("unknown method")
	ErrInvalidParamsData                      = errors.New("invalid params Data")
	ErrConfigStorageEmpty                     = errors.New("config storage is empty")
	ErrUnknownContract                        = errors.New("unknown contract")
	ErrNotTransferMethod                      = errors.New("not transfer method")
)
