package txmanager

import "errors"

var (
	// ErrChainNotSet reports Allocate/seed with no chain client wired via WithChain.
	ErrChainNotSet = errors.New("txmanager: no chain client is wired")
	// ErrStoreNotSet reports Allocate with no record store wired via WithStore.
	ErrStoreNotSet = errors.New("txmanager: no record store is wired")
)
