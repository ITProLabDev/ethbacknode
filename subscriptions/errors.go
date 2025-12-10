package subscriptions

import "errors"

var (
	ErrConfigStorageEmpty = errors.New("config storage not set")
	ErrUnknownTransaction = errors.New("unknown transaction")
	ErrUnknownServiceId   = errors.New("unknown serviceId")
)
