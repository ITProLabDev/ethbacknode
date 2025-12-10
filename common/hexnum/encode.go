package hexnum

import (
	"fmt"
	"math/big"
	"strings"
)

func Int64ToHex(num int64) string {
	return "0x" + fmt.Sprintf("%x", num)
}

func IntToHex(num int) string {
	return "0x" + fmt.Sprintf("%x", num)
}

func Uint64ToHex(num uint64) string {
	return "0x" + fmt.Sprintf("%x", num)
}

func UintToHex(num uint) string {
	return "0x" + fmt.Sprintf("%x", num)
}

func BigIntToHex(num *big.Int) string {
	if num == nil {
		return "0x0"
	}
	if num.Cmp(big.NewInt(0)) == 0 {
		return "0x0"
	}
	return "0x" + strings.TrimPrefix(fmt.Sprintf("%x", num.Bytes()), "0")
}

func BytesToHex(b []byte) string {
	return "0x" + fmt.Sprintf("%x", b)
}
