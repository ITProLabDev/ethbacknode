package hexnum

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
)

// ParseHexInt parse hex string value to int
func ParseHexInt(value string) (int, error) {
	i, err := strconv.ParseInt(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

// ParseHexInt64 parse hex string value to int64
func ParseHexInt64(value string) (int64, error) {
	i, err := strconv.ParseInt(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

// ParseHexUint64 parse hex string value to uint64
func ParseHexUint64(value string) (uint64, error) {
	i, err := strconv.ParseUint(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

// ParseBigInt parse hex string value to big.Int
func ParseBigInt(value string) (*big.Int, error) {
	value = strings.TrimPrefix(value, "0x")
	if len(value)%2 != 0 {
		value = "0" + value
	}
	b, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	i := &big.Int{}
	i.SetBytes(b)
	return i, err
}

func ParseHexBytes(value string) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(value, "0x"))
}
