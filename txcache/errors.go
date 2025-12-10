package txcache

import "errors"

var (
	ErrConfigStorageEmpty = errors.New("config storage is empty")
	ErrUnknownTransaction = errors.New("unknown transaction")
)
