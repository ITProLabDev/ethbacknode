package watchdog

import "errors"

var (
	ErrConfigStorageEmpty = errors.New("config storage not set")
	ErrAddressPoolNotSet  = errors.New("address pool not set")
	ErrChainClientNotSet  = errors.New("blockchain client not set")
)
