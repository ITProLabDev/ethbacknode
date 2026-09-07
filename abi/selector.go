package abi

import "github.com/ITProLabDev/ethbacknode/crypto"

// Selector returns the 4-byte method/event selector for a canonical ABI
// signature string (e.g. "transfer(address,uint256)") -- the same
// keccak256-based computation SmartContractAbiEntry.updateSignature uses
// internally, exposed so a caller can compute one without an ABI entry to
// hang it on (e.g. a contractSubscribe selector filter naming a method by its
// full signature rather than a raw hex selector).
//
// Two different signatures never collide here even when they share a bare
// name -- transfer(address,uint256) and transfer(address,uint256,bytes) hash
// differently -- which is why a selector filter should be built from full
// signatures (or raw selectors), never from a bare method name alone.
func Selector(canonicalSignature string) [4]byte {
	h := crypto.Keccak256([]byte(canonicalSignature))
	var out [4]byte
	copy(out[:], h[:4])
	return out
}
