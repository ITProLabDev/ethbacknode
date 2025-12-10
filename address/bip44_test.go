package address

import (
	"fmt"
	"module github.com/ITProLabDev/ethbacknode/common/bip32"
	"module github.com/ITProLabDev/ethbacknode/common/bip39"
	"module github.com/ITProLabDev/ethbacknode/common/bip44"
	"module github.com/ITProLabDev/ethbacknode/common/hexnum"
	"module github.com/ITProLabDev/ethbacknode/crypto"
	"module github.com/ITProLabDev/ethbacknode/tools/log"
	"strings"
	"testing"
)

type MockAddressCodec struct {
}

func (a *MockAddressCodec) EncodeBytesToAddress(addressBytes []byte) (string, error) {
	if len(addressBytes) != 20 {
		return "", ErrInvalidAddressBytes
	}
	addr := hexnum.BytesToHex(addressBytes)
	return _addressCheckSum(addr), nil
}

func (a *MockAddressCodec) DecodeAddressToBytes(addressStr string) ([]byte, error) {
	addressBytes, err := hexnum.ParseHexBytes(addressStr)
	if err != nil {
		return nil, ErrInvalidAddress
	}
	return addressBytes, nil

}

func (a *MockAddressCodec) PrivateKeyToAddress(privateKey []byte) (address string, addressBytes []byte, err error) {
	_, pub := crypto.ECDSAKeysFromPrivateKeyBytes(privateKey)
	pubKeyBytes := crypto.ECDSAPublicKeyToBytes(pub)
	addressBytes = crypto.Keccak256(pubKeyBytes[1:])[12:]
	address, _ = a.EncodeBytesToAddress(addressBytes)
	return address, addressBytes, nil
}

func (a *MockAddressCodec) IsValid(address string) bool {
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

func TestBip44Address(t *testing.T) {
	mockAddressCodec := &MockAddressCodec{}

	bitLen := 128

	entropy, err := bip39.NewEntropy(bitLen)

	fmt.Println("entropy:", hexnum.BytesToHex(entropy))

	mnemonicStr, err := bip39.NewMnemonic(entropy)

	mnemonic := strings.Split(mnemonicStr, " ")

	if err != nil {
		t.Error("bip39.NewMnemonic fail:", err)
	}

	seed := bip39.NewSeed(mnemonicStr, "")

	fmt.Println("seed:", hexnum.BytesToHex(seed))

	t.Log(mnemonic)
	masterKey, err := bip32.NewMasterKey(seed)
	fmt.Println("masterKey:", masterKey.String())

	key, err := bip44.NewKeyFromMasterKey(masterKey, bip44.TypeEther, 0x80000000, 0, 0)
	if err != nil {
		t.Error("bip44.NewKeyFromMasterKey fail:", err)
	}
	privateKey := key.Key
	if err != nil {
		t.Error("key.Serialize fail:", err)
	}
	fmt.Println("privateKey:", key.String())
	fmt.Println("privateKey:", hexnum.BytesToHex(privateKey))
	addressStr, addressBytes, err := mockAddressCodec.PrivateKeyToAddress(privateKey)
	if err != nil {
		t.Error("MockAddressCodec.PrivateKeyToAddress fail:", err)
	}
	addressRecord := &Address{
		Address:       addressStr,
		AddressBytes:  addressBytes,
		PrivateKey:    privateKey,
		Bip39Support:  true,
		Bip39Mnemonic: mnemonic,
	}
	log.Dump(addressRecord)
	ar2, err := createNewBIP44Address(12, bip44.TypeEther, mockAddressCodec)
	if err != nil {
		t.Error("createNewBIP44Address fail:", err)
	}
	log.Dump(ar2)
	log.Warning("Mnemonic:", strings.Join(ar2.Bip39Mnemonic, " "))
	arRestored, err := recoverBIP44AddressFromMnemonic(ar2.Bip39Mnemonic, bip44.TypeEther, mockAddressCodec)
	if err != nil {
		t.Error("recoverBIP44AddressFromMnemonic fail:", err)
	}
	log.Dump(arRestored)
	if arRestored.Address != ar2.Address {
		t.Error("Restored address not equal to original")
	}
}
