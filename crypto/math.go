package crypto

import "math/big"

// Constants for big.Int byte operations.
const (
	// wordBits is the number of bits in a big.Word.
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// wordBytes is the number of bytes in a big.Word.
	wordBytes = wordBits / 8
)

// paddedBigBytes returns the big.Int as a big-endian byte slice padded to n bytes.
func paddedBigBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	ret := make([]byte, n)
	readBits(bigint, ret)
	return ret
}

// ReadBits encodes the absolute value of bigint as big-endian bytes. Callers must ensure
// that buf has enough space. If buf is too short the result will be incomplete.
func readBits(bigint *big.Int, buf []byte) {
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < wordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
}
