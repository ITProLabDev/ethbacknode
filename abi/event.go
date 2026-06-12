package abi

import "github.com/ITProLabDev/ethbacknode/crypto"

// Topic0 returns the 32-byte event signature hash
// keccak256("EventName(type1,type2,...)") used as topics[0] for a
// non-anonymous event. Unlike a method selector (first 4 bytes), an event
// topic uses the full 32-byte hash.
func (e *SmartContractAbiEntry) Topic0() [32]byte {
	h := crypto.Keccak256([]byte(e.canonicalSignature()))
	var out [32]byte
	copy(out[:], h)
	return out
}
