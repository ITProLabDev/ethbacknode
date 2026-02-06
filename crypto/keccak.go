// Package crypto provides cryptographic operations for Ethereum transactions.
package crypto

import (
	"hash"

	"golang.org/x/crypto/sha3"
)

// Keccak256 calculates and returns the Keccak256 hash of the input data or few pieces of data.
func Keccak256(data ...[]byte) []byte {
	b := make([]byte, 32)
	d := NewKeccakState()
	for _, b := range data {
		_, _ = d.Write(b)
	}
	_, _ = d.Read(b)
	return b
}

// KeccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
// imported from go-ethereum, modified to remove dependency on go-ethereum/common because
// "A little copying is better than a little dependency." (https://go-proverbs.github.io) ;)
type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// NewKeccakState creates a new Keccak-256 hash state.
func NewKeccakState() KeccakState {
	return sha3.NewLegacyKeccak256().(KeccakState)
}
