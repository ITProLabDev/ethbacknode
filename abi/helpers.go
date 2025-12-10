package abi

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"
	"strings"
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
	for i, _ := range dst {
		dst[i] = padByte
	}
	copy(dst[bytesLen-len(src):], src)
	return dst
}

func bytesTrimZero(src []byte) (dst []byte) {
	var i int
	var b byte
	for i, b = range src {
		if b != 0 {
			break
		}
	}
	if len(src) == i {
		return []byte{}
	}
	dst = make([]byte, len(src)-i)
	copy(dst[0:], src[i:])
	return dst
}

// _parseHexInt parse hex string value to int
func _parseHexInt(value string) (int, error) {
	i, err := strconv.ParseInt(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return int(i), nil
}

// _parseHexInt64 parse hex string value to int64
func _parseHexInt64(value string) (int64, error) {
	i, err := strconv.ParseInt(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

// _parseBigInt parse hex string value to big.Int
func _parseBigInt(value string) (big.Int, error) {
	value = strings.TrimPrefix(value, "0x")
	i := big.Int{}
	_, err := fmt.Sscan(value, &i)
	return i, err
}

// _intToHex convert int to hexadecimal representation
func _intToHex(i int) string {
	return fmt.Sprintf("0x%x", i)
}
