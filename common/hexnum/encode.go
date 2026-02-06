// Package hexnum provides hex encoding/decoding utilities for numeric types.
// All hex strings use the "0x" prefix convention used in Ethereum.
package hexnum

import (
	"fmt"
	"math/big"
	"strings"
)

// Int64ToHex converts an int64 to a hex string with "0x" prefix.
func Int64ToHex(num int64) string {
	return "0x" + fmt.Sprintf("%x", num)
}

// IntToHex converts an int to a hex string with "0x" prefix.
func IntToHex(num int) string {
	return "0x" + fmt.Sprintf("%x", num)
}

// Uint64ToHex converts a uint64 to a hex string with "0x" prefix.
func Uint64ToHex(num uint64) string {
	return "0x" + fmt.Sprintf("%x", num)
}

// UintToHex converts a uint to a hex string with "0x" prefix.
func UintToHex(num uint) string {
	return "0x" + fmt.Sprintf("%x", num)
}

// BigIntToHex converts a big.Int to a hex string with "0x" prefix.
func BigIntToHex(num *big.Int) string {
	if num == nil {
		return "0x0"
	}
	if num.Cmp(big.NewInt(0)) == 0 {
		return "0x0"
	}
	return "0x" + strings.TrimPrefix(fmt.Sprintf("%x", num.Bytes()), "0")
}

// BytesToHex converts a byte slice to a hex string with "0x" prefix.
func BytesToHex(b []byte) string {
	return "0x" + fmt.Sprintf("%x", b)
}
