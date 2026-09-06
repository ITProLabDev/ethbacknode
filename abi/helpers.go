package abi

import (
	"encoding/binary"
)

func _byteToInt64(b []byte) int64 {
	b = bytePad(b, 8, 0)
	return int64(binary.BigEndian.Uint64(b))
}

func _int64ToByte(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func bytePad(src []byte, bytesLen int, padByte byte) []byte {
	dst := make([]byte, bytesLen)
	if padByte != 0 {
		for i := range dst {
			dst[i] = padByte
		}
	}
	if len(src) >= bytesLen {
		copy(dst, src[len(src)-bytesLen:])
		return dst
	}
	copy(dst[bytesLen-len(src):], src)
	return dst
}
