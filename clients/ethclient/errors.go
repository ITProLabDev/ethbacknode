package ethclient

import "errors"

var (
	ErrHashesOnlyBlockHash          = errors.New("only transactions hashes requested")
	ErrTransactionNotFound          = errors.New("transaction not found")
	ErrInvalidAddressCheckSum       = errors.New("invalid address checksum")
	ErrInvalidAddress               = errors.New("invalid address")
	ErrTransactionNotTransfer       = errors.New("transaction not transfer")
	ErrUnsupportedTransactionFormat = errors.New("unsupported transaction format")
	ErrUnsupportedTransactionType   = errors.New("unsupported transaction type")
	ErrTransactionSignError         = errors.New("transaction sign error")
	ErrInsufficientFunds            = errors.New("insufficient funds")
	ErrNothingToTransfer            = errors.New("nothing to transfer")
	ErrConfigStorageEmpty           = errors.New("config storage is empty")
	ErrUnknownToken                 = errors.New("unknown token")
)
