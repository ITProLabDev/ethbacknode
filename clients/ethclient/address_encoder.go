package ethclient

import (
	"module github.com/ITProLabDev/ethbacknode/address"
	"module github.com/ITProLabDev/ethbacknode/common/hexnum"
	"module github.com/ITProLabDev/ethbacknode/crypto"
	"strings"
)

func GetAddressCodec() address.AddressCodec {
	return &AddressCodec{}
}

type AddressCodec struct {
}

func (a *AddressCodec) EncodeBytesToAddress(addressBytes []byte) (string, error) {
	if len(addressBytes) != 20 {
		return "", address.ErrInvalidAddressBytes
	}
	addr := hexnum.BytesToHex(addressBytes)
	return _addressCheckSum(addr), nil
}

func (a *AddressCodec) DecodeAddressToBytes(addressStr string) ([]byte, error) {
	addressBytes, err := hexnum.ParseHexBytes(addressStr)
	if err != nil {
		return nil, address.ErrInvalidAddress
	}
	return addressBytes, nil

}

func (a *AddressCodec) PrivateKeyToAddress(privateKey []byte) (address string, addressBytes []byte, err error) {
	_, pub := crypto.ECDSAKeysFromPrivateKeyBytes(privateKey)
	pubKeyBytes := crypto.ECDSAPublicKeyToBytes(pub)
	addressBytes = crypto.Keccak256(pubKeyBytes[1:])[12:]
	address, _ = a.EncodeBytesToAddress(addressBytes)
	return address, addressBytes, nil
}

func (a *AddressCodec) IsValid(address string) bool {
	_, err := a.DecodeAddressToBytes(address)
	return err == nil
}

func _addressCheckSum(address string) string {
	hex := strings.ToLower(address)[2:]
	hash := crypto.Keccak256([]byte(hex))
	ret := "0x"
	for i, b := range hex {
		c := string(b)
		if b < '0' || b > '9' {
			if hash[i/2]&byte(128-i%2*120) != 0 {
				c = string(b - 32)
			}
		}
		ret += c
	}
	return ret
}
