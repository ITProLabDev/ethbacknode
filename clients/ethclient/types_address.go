package ethclient

import (
	"github.com/ITProLabDev/ethbacknode/crypto"
	"regexp"
	"strings"
)

type EthAddress string

var (
	isEthereumAddressValid = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
)

func (a EthAddress) IsValidate() bool {
	return addressIsValid(a)
}

func addressIsValid(address EthAddress) bool {
	if !isEthereumAddressValid.MatchString(string(address)) {
		return false
	}
	return true
}

// addressCheckSumIsValid check is "checksumed" address not changed.
// in Bitcoin or Tron used base58check encoding for ensure address is not corrupted
// while typing or copying. Ordinary Ethereum address is raw HEX encoded 20 bytes array
// and it's not possible to check is it correct or not. But Ethereum has a special
// checksum encoding format for addresses, added later.
func addressCheckSumIsValid(address string) (err error) {
	checksumed := addressCheckSumCreate(address)
	if address != checksumed {
		return ErrInvalidAddressCheckSum
	}
	return nil
}

// addressCheckSumCreate create address checksum.
// in Bitcoin or Tron used base58check encoding for ensure address is not corrupted
// while typing or copying. Ordinary Ethereum address is raw HEX encoded 20 bytes array
// and it's not possible to check is it correct or not. But Ethereum has a special
// checksum encoding format for addresses, added later.
func addressCheckSumCreate(address string) string {
	hex := strings.ToLower(address)[2:]
	hash := crypto.Keccak256([]byte(hex))
	checksumed := "0x"
	for i, b := range hex {
		c := string(b)
		if b < '0' || b > '9' {
			if hash[i/2]&byte(128-i%2*120) != 0 {
				c = string(b - 32)
			}
		}
		checksumed += c
	}
	return checksumed
}
