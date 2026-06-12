package eventlog

import "github.com/ITProLabDev/ethbacknode/abi"

// ContractEvent is a decoded contract event plus the block/tx context needed
// for delivery. It is the unit handed to a Sink.
type ContractEvent struct {
	Event            *abi.DecodedEvent // decoded event (name, contract, inputs)
	BlockNumber      int64
	TransactionHash  string
	TransactionIndex int64
	LogIndex         int64
	// Removed is true if the source log was reverted by a chain reorg.
	Removed bool
}

// ManagedAddresses is the subset of address.Manager that eventlog needs to
// decide managed_only matches. Satisfied by *address.Manager. Exported so
// main.go (another package) can wire it via WithManaged without friction.
type ManagedAddresses interface {
	IsAddressKnown(addr string) bool
}

// AddressEncoder turns a 20-byte address into its string form. Satisfied by
// the ethclient address codec. Exported for the same cross-package reason.
type AddressEncoder interface {
	EncodeBytesToAddress(b []byte) (string, error)
}

// eventMatchesScope reports whether a decoded event should be delivered to a
// subscription with the given scope. whole_contract always matches.
// managed_only matches iff at least one address-typed parameter of the event
// is a managed address.
func eventMatchesScope(ev *abi.DecodedEvent, scope Scope, managed ManagedAddresses, codec AddressEncoder) bool {
	if scope == ScopeWholeContract {
		return true
	}
	// ScopeManagedOnly: look for any address-typed input that is managed.
	for _, in := range ev.Inputs {
		if in.Type != "address" {
			continue
		}
		b, ok := in.Value.([]byte)
		if !ok {
			continue
		}
		addr, err := codec.EncodeBytesToAddress(b)
		if err != nil {
			continue
		}
		if managed.IsAddressKnown(addr) {
			return true
		}
	}
	return false
}
